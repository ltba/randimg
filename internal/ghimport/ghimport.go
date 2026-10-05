// Package ghimport GitHub 仓库导入: 枚举仓库文件树, 按拼接规则生成图源地址批量入库.
// 模块定义见 .trellis/spec/arch/ghimport.md; 需求真源见 prd F-012.
package ghimport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"randimg/internal/metadata"
	"randimg/internal/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	insertBatch = 500 // 每事务分批入库的上限组 (SQLite 变量数约束内查询)
	ghAPIBase   = "https://api.github.com"
	sampleCount = 3   // 预览返回的样例 URL 数
	probeCount  = 5   // 预览抽检的 HEAD 数
	probeBudget = 8 * time.Second // 抽检整体超时
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
	RepoURL    string `json:"repo_url"` // 完整仓库链接或 owner/repo, 自动解析
	Owner      string `json:"owner"`    // 显式覆盖 (兼容旧参数)
	Repo       string `json:"repo"`     // 显式覆盖
	Path       string `json:"path"`
	Ref        string `json:"ref"`
	BaseURL    string `json:"base_url"`
	CategoryID uint   `json:"category_id" binding:"required"`
	AutoFetch  *bool  `json:"auto_fetch"`
}

// parseRepoURL 解析仓库链接为 owner/repo/ref/path.
// 支持: https://github.com/o/r, https://github.com/o/r/tree/<ref>/<path>,
// github.com/o/r..., o/r. tree 段同时带出 ref 与 path.
func parseRepoURL(raw string) (owner, repo, ref, subPath string, err error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.Trim(s, "/")

	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", "", fmt.Errorf("invalid repo url: %s", raw)
	}
	owner, repo = parts[0], parts[1]
	repo = strings.TrimSuffix(repo, ".git")
	if len(parts) >= 4 && parts[2] == "tree" {
		ref = parts[3]
		if len(parts) > 4 {
			subPath = strings.Join(parts[4:], "/")
		}
	}
	return owner, repo, ref, subPath, nil
}

// resolveInput 合并 repo_url 解析结果与显式字段 (显式优先).
func resolveInput(in *importInput) error {
	if in.RepoURL != "" {
		owner, repo, ref, subPath, err := parseRepoURL(in.RepoURL)
		if err != nil {
			return err
		}
		if in.Owner == "" {
			in.Owner = owner
		}
		if in.Repo == "" {
			in.Repo = repo
		}
		if in.Ref == "" {
			in.Ref = ref
		}
		if in.Path == "" {
			in.Path = subPath
		}
	}
	if in.Owner == "" || in.Repo == "" {
		return fmt.Errorf("repo_url or owner/repo required")
	}
	if in.Ref == "" {
		in.Ref = "main"
	}
	in.Path = strings.Trim(in.Path, "/")
	return nil
}

// validateBaseURL 校验自定义图源地址形态.
func validateBaseURL(base string) error {
	if base == "" {
		return nil
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return fmt.Errorf("base_url must start with http:// or https://")
	}
	return nil
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
	if err := resolveInput(&in); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateBaseURL(in.BaseURL); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	urls := h.collectImageURLs(entries, base, in.Path)

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

// PreviewImport POST /api/admin/import/github/preview — 防呆预览:
// 枚举 tree, 返回匹配数, 样例完整 URL, 抽检可达性结果.
func (h *Handler) PreviewImport(c *gin.Context) {
	var in importInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := resolveInput(&in); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateBaseURL(in.BaseURL); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entries, err := h.listTree(c, in.Owner, in.Repo, in.Ref)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	base := strings.TrimRight(in.BaseURL, "/")
	if base == "" {
		base = fmt.Sprintf("%s/%s/%s/%s", "https://raw.githubusercontent.com", in.Owner, in.Repo, in.Ref)
	}
	urls := h.collectImageURLs(entries, base, in.Path)

	// 样例: 均匀抽取.
	samples := make([]string, 0, sampleCount)
	if len(urls) > 0 {
		step := len(urls) / sampleCount
		if step < 1 {
			step = 1
		}
		for i := 0; i < len(urls) && len(samples) < sampleCount; i += step {
			samples = append(samples, urls[i])
		}
	}

	// 抽检: 随机取 probeCount 张并发 HEAD.
	probes := pickProbes(urls, probeCount)
	checks := h.probeURLs(c.Request.Context(), probes)
	okCount := 0
	for _, chk := range checks {
		if chk["ok"] == true {
			okCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matched":  len(urls),
		"samples":  samples,
		"checks":   checks,
		"probe_ok": okCount,
		"probe_total": len(probes),
	})
}

// collectImageURLs 过滤图片扩展名与 path 前缀, 拼接完整图源地址.
func (h *Handler) collectImageURLs(entries []treeEntry, base, subPath string) []string {
	prefix := ""
	if subPath != "" {
		prefix = subPath + "/"
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
	return urls
}

// pickProbes 均匀抽样 n 张 (确定性, 避免引入随机源).
func pickProbes(urls []string, n int) []string {
	if len(urls) <= n {
		return urls
	}
	out := make([]string, 0, n)
	step := len(urls) / n
	for i := 0; i < len(urls) && len(out) < n; i += step {
		out = append(out, urls[i])
	}
	return out
}

// probeURLs 并发 HEAD 抽检, 单张 5s 超时, 整体受限.
type probeResult struct {
	url    string
	ok     bool
	detail string
}

func (h *Handler) probeURLs(ctx context.Context, urls []string) []gin.H {
	var mu sync.Mutex
	results := make([]probeResult, len(urls))
	var wg sync.WaitGroup
	probeClient := &http.Client{Timeout: 5 * time.Second}
	ctx, cancel := context.WithTimeout(ctx, probeBudget)
	defer cancel()

	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
			if err != nil {
				results[i] = probeResult{u, false, err.Error()}
				return
			}
			resp, err := probeClient.Do(req)
			if err != nil {
				mu.Lock()
				results[i] = probeResult{u, false, err.Error()}
				mu.Unlock()
			return
		}
			resp.Body.Close()
			ok := resp.StatusCode >= 200 && resp.StatusCode < 400
			detail := fmt.Sprintf("HTTP %d", resp.StatusCode)
			if !ok && resp.StatusCode == http.StatusNotImplemented {
				// 个别源不支持 HEAD, 用 GET 兜底确认.
				greq, gerr := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if gerr == nil {
					gresp, gerr := probeClient.Do(greq)
					if gerr == nil {
						gresp.Body.Close()
						ok = gresp.StatusCode >= 200 && gresp.StatusCode < 400
						detail = fmt.Sprintf("HTTP %d (GET)", gresp.StatusCode)
					}
				}
			}
			mu.Lock()
			results[i] = probeResult{u, ok, detail}
			mu.Unlock()
		}(i, u)
	}
	wg.Wait()

	out := make([]gin.H, len(results))
	for i, r := range results {
		out[i] = gin.H{"url": r.url, "ok": r.ok, "detail": r.detail}
	}
	return out
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
