<template>
  <section>
    <div class="page-head">
      <h2>分类管理</h2>
      <button class="btn btn-primary" @click="openEditor(null)">添加分类</button>
    </div>
    <table>
      <thead>
        <tr>
          <th>ID</th>
          <th>名称</th>
          <th>Slug</th>
          <th>描述</th>
          <th>图片数</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="cat in categories" :key="cat.id">
          <td class="num">{{ cat.id }}</td>
          <td>{{ cat.name }}</td>
          <td><code>{{ cat.slug }}</code></td>
          <td>{{ cat.description || '-' }}</td>
          <td class="num">
            <span v-if="counts[cat.id] === undefined">…</span>
            <template v-else>
              {{ counts[cat.id] }}
              <span v-if="counts[cat.id] > 0" class="text-2">(删除保护)</span>
            </template>
          </td>
          <td>
            <button class="btn btn-sm btn-primary" @click="openEditor(cat)">编辑</button>
            <button class="btn btn-sm btn-danger" @click="removeCategory(cat)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <ModalShell :open="editorOpen" :title="editorId ? '编辑分类' : '添加分类'" @close="editorOpen = false">
      <form @submit.prevent="saveEditor">
        <div class="form-group">
          <label>名称 *</label>
          <input v-model="editor.name" type="text" required />
        </div>
        <div class="form-group">
          <label>Slug *</label>
          <input v-model="editor.slug" type="text" required placeholder="acg, landscape, etc." />
        </div>
        <div class="form-group">
          <label>描述</label>
          <textarea v-model="editor.description"></textarea>
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary">保存</button>
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">取消</button>
        </div>
      </form>
    </ModalShell>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import type { Category } from '../../shared/types';
import { adminRequest } from '../../shared/api';
import { toast, confirmModal } from '../ui';
import ModalShell from '../components/ModalShell.vue';

const categories = ref<Category[]>([]);
const counts = ref<Record<number, number>>({});

const editorOpen = ref(false);
const editorId = ref<number | null>(null);
const editor = reactive({ name: '', slug: '', description: '' });

onMounted(() => load());

async function load() {
  try {
    categories.value = await adminRequest<Category[]>('/categories');
    const result: Record<number, number> = {};
    await Promise.all(
      categories.value.map(async cat => {
        const data = await adminRequest<{ pagination: { total: number } }>(
          `/images?category=${cat.slug}&page_size=1`,
        );
        result[cat.id] = data.pagination.total;
      }),
    );
    counts.value = result;
  } catch (error) {
    toast('加载分类失败: ' + (error as Error).message, 'error');
  }
}

function openEditor(cat: Category | null) {
  editorId.value = cat?.id ?? null;
  editor.name = cat?.name ?? '';
  editor.slug = cat?.slug ?? '';
  editor.description = cat?.description ?? '';
  editorOpen.value = true;
}

async function saveEditor() {
  const payload = { name: editor.name, slug: editor.slug, description: editor.description || null };
  try {
    if (editorId.value) {
      await adminRequest(`/categories/${editorId.value}`, { method: 'PUT', body: JSON.stringify(payload) });
      toast('分类更新成功');
    } else {
      await adminRequest('/categories', { method: 'POST', body: JSON.stringify(payload) });
      toast('分类添加成功');
    }
    editorOpen.value = false;
    load();
  } catch (error) {
    toast('操作失败: ' + (error as Error).message, 'error');
  }
}

async function removeCategory(cat: Category) {
  const count = counts.value[cat.id];
  if (count !== undefined && count > 0) {
    toast(`分类「${cat.name}」下有 ${count} 张图片, 无法删除; 请先转移图片`, 'error');
    return;
  }
  if (!(await confirmModal(`确定要删除分类「${cat.name}」吗？`))) return;
  try {
    await adminRequest(`/categories/${cat.id}`, { method: 'DELETE' });
    toast('分类删除成功');
    load();
  } catch (error) {
    toast('删除失败: ' + (error as Error).message, 'error');
  }
}
</script>
