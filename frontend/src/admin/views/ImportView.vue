<template>
  <section>
    <div class="page-head"><h2>批量导入</h2></div>
    <p class="section-hint">粘贴图片 URL 列表 (每行一个, 单次最多 1000 条, 重复行自动去重)</p>
    <form @submit.prevent="batchImport">
      <div class="form-group">
        <label>图片 URL 列表 *</label>
        <textarea v-model="batchUrls" required placeholder="https://example.com/1.jpg&#10;https://example.com/2.jpg"></textarea>
      </div>
      <div class="form-group">
        <label>分类 *</label>
        <select v-model.number="batchCategoryId" required>
          <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
      </div>
      <div class="form-group">
        <label>
          <input v-model="batchAutoFetch" type="checkbox" />
          自动获取图片信息（尺寸和格式）
        </label>
      </div>
      <p v-if="batchHint" class="section-hint">{{ batchHint }}</p>
      <button type="submit" class="btn btn-primary">导入</button>
    </form>

    <h2 class="mt-20">GitHub 仓库导入</h2>
    <p class="section-hint">粘贴仓库链接, 自动解析 owner / repo / 分支 / 子目录; 先预览确认再导入</p>
    <form @submit.prevent="githubImport">
      <div class="form-group">
        <label>仓库链接 *</label>
        <input v-model="gh.repoUrl" type="text" required placeholder="https://github.com/owner/repo/tree/main/images" />
        <small>支持完整 URL, github.com/owner/repo, 或 owner/repo; 含 /tree/&lt;分支&gt;/&lt;子目录&gt; 时自动带出</small>
      </div>
      <div class="form-group">
        <label>图源模式</label>
        <div class="radio-row">
          <label><input v-model="gh.mode" type="radio" value="github" /> GitHub raw (默认)</label>
          <label><input v-model="gh.mode" type="radio" value="mirror" /> 自定义镜像</label>
        </div>
      </div>
      <div v-if="gh.mode === 'mirror'" class="form-group">
        <label>镜像 Base URL *</label>
        <input v-model="gh.baseUrl" type="url" placeholder="https://mirror.example.com/repo" />
        <small>图源地址 = Base URL + 仓库内路径; 末尾斜杠自动忽略</small>
      </div>
      <div class="form-group">
        <label>分类 *</label>
        <select v-model.number="gh.categoryId" required>
          <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
      </div>
      <div class="form-group">
        <label>
          <input v-model="gh.autoFetch" type="checkbox" />
          入库后自动获取图片信息
        </label>
      </div>
      <details class="advanced">
        <summary>高级选项 (覆盖自动解析结果)</summary>
        <div class="form-group">
          <label>子目录</label>
          <input v-model="gh.path" type="text" placeholder="留空使用 URL 解析结果" />
        </div>
        <div class="form-group">
          <label>分支 / Ref</label>
          <input v-model="gh.ref" type="text" placeholder="留空使用 URL 解析结果, 缺省 main" />
        </div>
      </details>

      <div v-if="preview" class="preview-panel visible">
        <h4>匹配图片</h4>
        <div class="preview-count num">{{ preview.matched }}</div>
        <h4>样例图源 URL (点击复制)</h4>
        <ul class="sample-list">
          <li v-for="u in preview.samples" :key="u" :title="u" @click="copySample(u)">{{ u }}</li>
        </ul>
        <h4>抽检验证</h4>
        <div v-for="c in preview.checks" :key="c.url" :class="c.ok ? 'check-ok' : 'check-fail'">
          {{ c.ok ? '✓' : '✗' }} {{ c.detail }} — {{ c.url }}
        </div>
        <p v-if="preview.probe_total > 0 && preview.probe_ok === 0" class="section-hint check-fail">
          抽检全部失败, 图源地址大概率拼错, 已禁止导入
        </p>
        <p v-else-if="preview.probe_ok < preview.probe_total" class="section-hint check-fail">
          抽检 {{ preview.probe_ok }}/{{ preview.probe_total }} 通过, 请检查失败样例后再导入
        </p>
      </div>

      <div class="form-actions">
        <button type="button" class="btn btn-secondary" :disabled="previewing" @click="runPreview">
          {{ previewing ? '预览中...' : '预览' }}
        </button>
        <button type="submit" class="btn btn-primary" :disabled="!importAllowed">导入</button>
      </div>
      <p v-if="ghResult" class="section-hint">{{ ghResult }}</p>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import type {
  BatchCreateResult,
  Category,
  ImportPreview,
  ImportResult,
} from '../../shared/types';
import { adminRequest, copyText } from '../../shared/api';
import { toast } from '../ui';

defineProps<{ categories: Category[] }>();

const emit = defineEmits<{ changed: [] }>();

// ---- URL 列表批量导入 ----
const batchUrls = ref('');
const batchCategoryId = ref(0);
const batchAutoFetch = ref(true);
const batchHint = ref('');

async function batchImport() {
  const raw = batchUrls.value.split(/[\n,]/).map(s => s.trim()).filter(s => s !== '');
  if (raw.length === 0) {
    toast('请粘贴至少一个图片URL', 'error');
    return;
  }
  const unique = Array.from(new Set(raw));
  if (unique.length > 1000) {
    toast('单次最多 1000 条', 'error');
    return;
  }
  const dedupNote = unique.length < raw.length ? ` (自动去重 ${raw.length - unique.length} 条)` : '';
  try {
    const result = await adminRequest<BatchCreateResult>('/images/batch', {
      method: 'POST',
      body: JSON.stringify({
        images: unique.map(u => ({
          source_url: u,
          category_id: batchCategoryId.value,
          auto_fetch: batchAutoFetch.value,
        })),
      }),
    });
    toast(`批量导入成功！共添加 ${result.count} 张图片${dedupNote}`);
    batchHint.value = `已导入 ${result.count} 张${dedupNote} (元数据补全排队 ${result.fetch_pending ?? 0})`;
    batchUrls.value = '';
    emit('changed');
  } catch (error) {
    toast('批量导入失败: ' + (error as Error).message, 'error');
  }
}

// ---- GitHub 仓库导入 ----
const gh = reactive({
  repoUrl: '',
  mode: 'github' as 'github' | 'mirror',
  baseUrl: '',
  categoryId: 0,
  autoFetch: true,
  path: '',
  ref: '',
});

const preview = ref<ImportPreview | null>(null);
const previewing = ref(false);
const ghResult = ref('');

function ghPayload(): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    repo_url: gh.repoUrl.trim(),
    category_id: gh.categoryId,
    auto_fetch: gh.autoFetch,
  };
  if (gh.mode === 'mirror') payload.base_url = gh.baseUrl.trim();
  if (gh.path.trim()) payload.path = gh.path.trim();
  if (gh.ref.trim()) payload.ref = gh.ref.trim();
  return payload;
}

// 参数变化后预览失效.
watch(
  () => [gh.repoUrl, gh.mode, gh.baseUrl, gh.path, gh.ref] as const,
  () => {
    preview.value = null;
  },
);

const importAllowed = computed(() => {
  if (!preview.value || preview.value.matched === 0) return false;
  if (preview.value.probe_total > 0 && preview.value.probe_ok === 0) return false;
  return true;
});

async function runPreview() {
  if (!gh.repoUrl.trim()) {
    toast('请先填写仓库链接', 'error');
    return;
  }
  if (gh.mode === 'mirror' && !gh.baseUrl.trim()) {
    toast('自定义镜像需要填写 Base URL', 'error');
    return;
  }
  previewing.value = true;
  preview.value = null;
  try {
    preview.value = await adminRequest<ImportPreview>('/import/github/preview', {
      method: 'POST',
      body: JSON.stringify(ghPayload()),
    });
    if (preview.value.matched === 0) {
      toast('未匹配到图片, 请检查子目录或分支', 'error');
    }
  } catch (error) {
    toast('预览失败: ' + (error as Error).message, 'error');
  } finally {
    previewing.value = false;
  }
}

async function githubImport() {
  if (!importAllowed.value) return;
  ghResult.value = '导入中, 仓库较大时可能需要数十秒...';
  try {
    const result = await adminRequest<ImportResult>('/import/github', {
      method: 'POST',
      body: JSON.stringify(ghPayload()),
    });
    toast(`GitHub 导入完成: 新增 ${result.imported}, 跳过 ${result.skipped}`);
    ghResult.value = `新增 ${result.imported} 张, 跳过 ${result.skipped} 张 (已存在), 元数据补全排队 ${result.fetch_pending}`;
    preview.value = null;
    emit('changed');
  } catch (error) {
    ghResult.value = '';
    toast('GitHub 导入失败: ' + (error as Error).message, 'error');
  }
}

async function copySample(u: string) {
  if (await copyText(u)) toast('已复制');
}
</script>

<style scoped>
.preview-panel {
  margin-top: 14px;
  padding: 16px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.preview-panel h4 {
  font-size: 13px;
  color: var(--text-2);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin: 10px 0 6px;
}

.preview-panel h4:first-child {
  margin-top: 0;
}

.preview-count {
  font-size: 22px;
  font-weight: 700;
}

.sample-list {
  list-style: none;
}

.sample-list li {
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sample-list li:hover {
  background: var(--surface);
}

.check-ok {
  color: var(--ok-text);
  font-size: 13px;
  word-break: break-all;
}

.check-fail {
  color: #991b1b;
  font-size: 13px;
  word-break: break-all;
}

details.advanced {
  margin-top: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px 14px;
  background: var(--surface);
}

details.advanced summary {
  cursor: pointer;
  color: var(--text-2);
  font-size: 13px;
  user-select: none;
}

details.advanced[open] summary {
  margin-bottom: 12px;
}
</style>
