// Package ghimport GitHub 仓库导入: 枚举仓库文件树, 按拼接规则生成图源地址批量入库.
// 模块定义见 .trellis/spec/arch/ghimport.md; 需求真源见 prd F-012.
package ghimport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"randimg/internal/metadata"
	"randimg/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	insertBatch = 500 // 每事务分批入库的上限组 (SQLite 变量数约束内查询)
	ghAPIBase   = "https://api.github.com"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// Handler GitHub 导入端点.
type Handler struct {
	client *http.Client
	fetch  *metadata.MetadataFetchService
}

// NewHandler 构造导入 Handler; GitHub API 客户端与超时集中本模块.
func NewHandler(fetch *metadata.MetadataFetchService) *Handler {
	return &Handler{
		client: &http.Client{Timeout: 30 * time.Second},
		fetch:  fetch,
	}
}

type importInput struct {
	Owner      string `json:"owner" binding:"required"`
	Repo       string `json:"repo" binding:"required"`
	Path       string `json:"path"`
	Ref        string `json:"ref"`
	BaseURL    string `json:"base_url"`
	CategoryID uint   `json:"category_id" binding:"required"`
	AutoFetch  *bool  `json:"auto_fetch"`
}

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type treeResponse struct {
	Tree      []treeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

// ImportFromGitHub POST /api/admin/import/github
func (h *Handler) ImportFromGitHub(c *gin.Context) {
	var in importInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.Ref == "" {
		in.Ref = "main"
	}
	in.Path = strings.Trim(in.Path, "/")

	// 分类存在性.
	var cat store.Category
	if err := store.DB.First(&cat, in.CategoryID).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Category not found"})
		return
	}

	entries, err := h.listTree(c, in.Owner, in.Repo, in.Ref)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 拼接规则: base_url 可选, 缺省 raw.githubusercontent.com/<owner>/<repo>/<ref>.
	base := strings.TrimRight(in.BaseURL, "/")
	if base == "" {
		base = fmt.Sprintf("%s/%s/%s/%s", "https://raw.githubusercontent.com", in.Owner, in.Repo, in.Ref)
	}

	prefix := ""
	if in.Path != "" {
		prefix = in.Path + "/"
	}

	var urls []string
	for _, e := range entries {
		if e.Type != "blob" || !imageExts[strings.ToLower(path.Ext(e.Path))] {
			continue
		}
		if prefix != "" && !strings.HasPrefix(e.Path, prefix) {
			continue
		}
		urls = append(urls, base+"/"+e.Path)
	}

	// 幂等: 分批查询已存在图源地址, 跳过.
	existing := make(map[string]bool, len(urls))
	for i := 0; i < len(urls); i += insertBatch {
		end := min(i+insertBatch, len(urls))
		var found []string
		if err := store.DB.Model(&store.Image{}).
			Where("source_url IN ?", urls[i:end]).
			Pluck("source_url", &found).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing images"})
			return
		}
		for _, u := range found {
			existing[u] = true
		}
	}

	images := make([]store.Image, 0, len(urls))
	source := fmt.Sprintf("GitHub - %s/%s", in.Owner, in.Repo)
	for _, u := range urls {
		if existing[u] {
			continue
		}
		images = append(images, store.Image{
			SourceURL:  u,
			Source:     source,
			CategoryID: in.CategoryID,
			Status:     "active",
		})
	}

	imported := 0
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(images); i += insertBatch {
			end := min(i+insertBatch, len(images))
			slice := images[i:end]
			if err := tx.CreateInBatches(&slice, 100).Error; err != nil {
				return err
			}
			imported = end
		}
		return nil
	}); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to import images"})
		return
	}

	fetchPending := 0
	if in.AutoFetch == nil || *in.AutoFetch { // 缺省 true (F-012).
		for i := range images[:imported] {
			if h.fetch.Enqueue(images[i].ID) {
				fetchPending++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"imported":      imported,
		"skipped":       len(urls) - imported,
		"fetch_pending": fetchPending,
	})
}

// listTree 枚举仓库文件树 (recursive), 校验截断.
func (h *Handler) listTree(c *gin.Context, owner, repo, ref string) ([]treeEntry, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", ghAPIBase, owner, repo, ref)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github api status %d: %s", resp.StatusCode, string(body))
	}

	var tree treeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, fmt.Errorf("decode tree: %w", err)
	}
	if tree.Truncated {
		return nil, fmt.Errorf("repository tree truncated, use a narrower path")
	}
	return tree.Tree, nil
}
