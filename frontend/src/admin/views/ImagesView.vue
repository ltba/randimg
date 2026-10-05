<template>
  <section>
    <div class="page-head">
      <h2>图片管理</h2>
      <div class="table-tools">
        <input
          v-model="searchInput"
          class="search"
          type="search"
          placeholder="搜索 URL / 来源, 回车确认"
          @keydown.enter="applySearch"
          @blur="applySearch"
        />
        <select v-model="categoryFilter" @change="reload">
          <option value="">全部分类</option>
          <option v-for="c in categories" :key="c.id" :value="c.slug">{{ c.name }}</option>
        </select>
        <select v-model="statusFilter" @change="reload">
          <option value="">全部状态</option>
          <option value="active">激活</option>
          <option value="inactive">下线</option>
        </select>
        <select v-model="fetchFilter" @change="reload">
          <option value="">全部图片</option>
          <option value="true">补全失败</option>
        </select>
        <button class="btn btn-secondary" @click="fetchMissing">补全缺失元数据</button>
        <button class="btn btn-primary" @click="openEditor(null)">添加图片</button>
      </div>
    </div>

    <div v-if="selected.size > 0" class="batch-actions visible">
      <span style="margin-right: 15px">
        已选择 <strong>{{ selected.size }}</strong> 张图片
      </span>
      <button class="btn btn-sm btn-primary" @click="openBatchUpdate">批量修改</button>
      <button class="btn btn-sm btn-danger" @click="batchDelete">批量删除</button>
      <button class="btn btn-sm btn-secondary" @click="selected.clear()">取消选择</button>
    </div>

    <table>
      <thead>
        <tr>
          <th><input v-model="selectAll" type="checkbox" @change="toggleSelectAll" /></th>
          <th>ID</th>
          <th>预览</th>
          <th>URL</th>
          <th>尺寸</th>
          <th>分类</th>
          <th>补全</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="img in images" :key="img.id" class="row" @click="detail = img">
          <td @click.stop>
            <input
              v-model="selectedIds"
              :value="img.id"
              type="checkbox"
              class="image-checkbox"
            />
          </td>
          <td class="num">{{ img.id }}</td>
          <td><img :src="img.source_url" class="thumb" loading="lazy" alt="" /></td>
          <td class="cell-url" :title="`${img.source_url} (点击复制)`" @click.stop="copyUrl(img)">{{ img.source_url }}</td>
          <td class="num">{{ img.width && img.height ? `${img.width} x ${img.height}` : '-' }}</td>
          <td>{{ img.category?.name || '-' }}</td>
          <td>
            <span v-if="img.fetch_fails >= 3" class="pill pill-warn">失败 {{ img.fetch_fails }}</span>
            <span v-else-if="!img.width || !img.height" class="pill pill-off">待补全</span>
            <span v-else class="pill pill-ok">完整</span>
          </td>
          <td>
            <span class="pill" :class="img.status === 'active' ? 'pill-ok' : 'pill-off'">{{ img.status }}</span>
          </td>
          <td @click.stop>
            <button class="btn btn-sm btn-primary" @click="openEditor(img)">编辑</button>
            <button class="btn btn-sm btn-danger" @click="removeImage(img)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <PaginationBar
      :page="page"
      :page-size="pageSize"
      :total-page="totalPage"
      @go="p => load(p)"
      @size="onPageSize"
    />

    <ImageDrawer
      :image="detail"
      @close="detail = null"
      @edit="img => { detail = null; openEditor(img); }"
      @remove="img => { detail = null; removeImage(img); }"
    />

    <ModalShell :open="editorOpen" :title="editorId ? '编辑图片' : '添加图片'" @close="editorOpen = false">
      <form @submit.prevent="saveEditor">
        <div class="form-group">
          <label>图片URL *</label>
          <input v-model="editor.source_url" type="url" required />
        </div>
        <div class="form-group">
          <label>
            <input v-model="editor.auto_fetch" type="checkbox" />
            自动获取图片信息（尺寸和格式）
          </label>
        </div>
        <div class="form-group">
          <label>分类 *</label>
          <select v-model.number="editor.category_id" required>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>宽度</label>
          <input v-model.number="editor.width" type="number" />
        </div>
        <div class="form-group">
          <label>高度</label>
          <input v-model.number="editor.height" type="number" />
        </div>
        <div class="form-group">
          <label>格式</label>
          <input v-model="editor.format" type="text" placeholder="jpeg, png, webp" />
        </div>
        <div class="form-group">
          <label>来源</label>
          <input v-model="editor.source" type="text" placeholder="Unsplash, GitHub, etc." />
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary">保存</button>
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">取消</button>
        </div>
      </form>
    </ModalShell>

    <ModalShell :open="batchOpen" title="批量修改图片" @close="batchOpen = false">
      <p class="section-hint" style="margin-bottom: 14px">
        将对选中的 <strong>{{ selected.size }}</strong> 张图片进行修改
      </p>
      <form @submit.prevent="saveBatchUpdate">
        <div class="form-group">
          <label>分类</label>
          <select v-model="batchUpdates.category_id">
            <option value="">不修改</option>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>状态</label>
          <select v-model="batchUpdates.status">
            <option value="">不修改</option>
            <option value="active">激活</option>
            <option value="inactive">暂时下线</option>
          </select>
        </div>
        <div class="form-group">
          <label>来源</label>
          <input v-model="batchUpdates.source" type="text" placeholder="不填则不修改" />
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary">确认修改</button>
          <button type="button" class="btn btn-secondary" @click="batchOpen = false">取消</button>
        </div>
      </form>
    </ModalShell>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import type { Category, Image, ImageListResponse } from '../../shared/types';
import { adminRequest, copyText } from '../../shared/api';
import { toast, confirmModal } from '../ui';
import PaginationBar from '../components/PaginationBar.vue';
import ImageDrawer from '../components/ImageDrawer.vue';
import ModalShell from '../components/ModalShell.vue';

const emit = defineEmits<{ changed: [] }>();

const props = defineProps<{ categories: Category[] }>();

const images = ref<Image[]>([]);
const page = ref(1);
const pageSize = ref(20);
const totalPage = ref(0);

const searchInput = ref('');
const search = ref('');
const categoryFilter = ref('');
const statusFilter = ref('');
const fetchFilter = ref('');

const selectedIds = ref<number[]>([]);
const selected = computed(() => new Set(selectedIds.value));
const selectAll = ref(false);

const detail = ref<Image | null>(null);

const editorOpen = ref(false);
const editorId = ref<number | null>(null);
const editor = reactive({
  source_url: '',
  auto_fetch: true,
  category_id: 0,
  width: null as number | null,
  height: null as number | null,
  format: '',
  source: '',
});

const batchOpen = ref(false);
const batchUpdates = reactive({ category_id: '' as number | '', status: '', source: '' });

onMounted(() => load(1));

watch(selectedIds, () => {
  selectAll.value = images.value.length > 0 && selectedIds.value.length === images.value.length;
});

function applySearch() {
  if (searchInput.value.trim() !== search.value) {
    search.value = searchInput.value.trim();
    load(1);
  }
}

function reload() {
  load(1);
}

async function load(p: number) {
  try {
    const params = new URLSearchParams({ page: String(p), page_size: String(pageSize.value) });
    if (search.value) params.set('q', search.value);
    if (categoryFilter.value) params.set('category', categoryFilter.value);
    if (statusFilter.value) params.set('status', statusFilter.value);
    if (fetchFilter.value) params.set('fetch_failed', fetchFilter.value);
    const data = await adminRequest<ImageListResponse>(`/images?${params}`);
    images.value = data.data;
    page.value = data.pagination.page;
    totalPage.value = data.pagination.total_page;
  } catch (error) {
    toast('加载图片失败: ' + (error as Error).message, 'error');
  }
}

function onPageSize(size: number) {
  pageSize.value = size;
  load(1);
}

function toggleSelectAll() {
  selectedIds.value = selectAll.value ? images.value.map(i => i.id) : [];
}

async function copyUrl(img: Image) {
  if (await copyText(img.source_url)) {
    toast('已复制 URL');
  } else {
    toast('复制失败', 'error');
  }
}

function openEditor(img: Image | null) {
  editorId.value = img?.id ?? null;
  editor.source_url = img?.source_url ?? '';
  editor.auto_fetch = true;
  editor.category_id = img?.category_id ?? props.categories[0]?.id ?? 0;
  editor.width = img?.width ?? null;
  editor.height = img?.height ?? null;
  editor.format = img?.format ?? '';
  editor.source = img?.source ?? '';
  editorOpen.value = true;
}

async function saveEditor() {
  const payload: Record<string, unknown> = {
    source_url: editor.source_url,
    category_id: editor.category_id,
    width: editor.width,
    height: editor.height,
    format: editor.format || null,
    source: editor.source || null,
    auto_fetch: editor.auto_fetch,
  };
  try {
    if (editorId.value) {
      await adminRequest(`/images/${editorId.value}`, { method: 'PUT', body: JSON.stringify(payload) });
      toast('图片更新成功');
    } else {
      await adminRequest('/images', { method: 'POST', body: JSON.stringify(payload) });
      toast('图片添加成功');
    }
    editorOpen.value = false;
    load(page.value);
    emit('changed');
  } catch (error) {
    toast('操作失败: ' + (error as Error).message, 'error');
  }
}

async function removeImage(img: Image) {
  if (!(await confirmModal(`确定要删除图片 #${img.id} 吗？`))) return;
  try {
    await adminRequest(`/images/${img.id}`, { method: 'DELETE' });
    toast('图片删除成功');
    load(page.value);
    emit('changed');
  } catch (error) {
    toast('删除失败: ' + (error as Error).message, 'error');
  }
}

function openBatchUpdate() {
  batchUpdates.category_id = '';
  batchUpdates.status = '';
  batchUpdates.source = '';
  batchOpen.value = true;
}

async function saveBatchUpdate() {
  const updates: Record<string, unknown> = {};
  if (batchUpdates.category_id !== '') updates.category_id = batchUpdates.category_id;
  if (batchUpdates.status) updates.status = batchUpdates.status;
  if (batchUpdates.source) updates.source = batchUpdates.source;
  if (Object.keys(updates).length === 0) {
    toast('请至少选择一项要修改的内容', 'error');
    return;
  }
  try {
    const result = await adminRequest<{ updated: number }>('/images/batch', {
      method: 'PUT',
      body: JSON.stringify({ image_ids: Array.from(selected.value), updates }),
    });
    toast(`批量修改成功！已更新 ${result.updated} 张图片`);
    batchOpen.value = false;
    selectedIds.value = [];
    load(page.value);
  } catch (error) {
    toast('批量修改失败: ' + (error as Error).message, 'error');
  }
}

async function batchDelete() {
  if (selected.value.size === 0) return;
  if (!(await confirmModal(`确定要删除选中的 ${selected.value.size} 张图片吗？`))) return;
  try {
    const result = await adminRequest<{ deleted: number }>('/images/batch', {
      method: 'DELETE',
      body: JSON.stringify({ image_ids: Array.from(selected.value) }),
    });
    toast(`批量删除成功！已删除 ${result.deleted} 张图片`);
    selectedIds.value = [];
    load(page.value);
    emit('changed');
  } catch (error) {
    toast('批量删除失败: ' + (error as Error).message, 'error');
  }
}

defineExpose({ reload });

async function fetchMissing() {
  if (!(await confirmModal('对全部缺失元数据的图片重新发起补全？'))) return;
  try {
    const result = await adminRequest<{ queued: number }>('/images/auto-fetch', {
      method: 'POST',
      body: JSON.stringify({ all: true }),
    });
    toast(`已入队 ${result.queued} 张补全任务`);
  } catch (error) {
    toast('触发补全失败: ' + (error as Error).message, 'error');
  }
}
</script>

<style scoped>
.search {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  width: 220px;
}

.search:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-soft);
}

.table-tools select {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  background: var(--surface);
}

.row {
  cursor: pointer;
}
</style>
