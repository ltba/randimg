<template>
  <section>
    <div class="page-head">
      <h2>Channels</h2>
      <button class="btn btn-primary" @click="openEditor(null)">创建Channel</button>
    </div>
    <table>
      <thead>
        <tr>
          <th>ID</th>
          <th>Channel</th>
          <th>限流(次/分钟)</th>
          <th>来源绑定</th>
          <th>状态</th>
          <th>创建时间</th>
          <th>最后使用</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ch in channels" :key="ch.id">
          <td class="num">{{ ch.id }}</td>
          <td class="cell-channel" :title="`${ch.channel_id} (点击复制)`" @click="copyChannel(ch)">
            {{ copiedId === ch.channel_id ? '已复制!' : ch.channel_id }}
          </td>
          <td class="num">{{ ch.rate_limit }}</td>
          <td class="cell-origins" :title="originsText(ch)">{{ originsText(ch) }}</td>
          <td>
            <span class="pill" :class="ch.status === 'active' ? 'pill-ok' : 'pill-off'">{{ ch.status }}</span>
          </td>
          <td>{{ formatDate(ch.created_at) }}</td>
          <td>{{ formatDate(ch.last_used_at) }}</td>
          <td>
            <button class="btn btn-sm btn-primary" @click="openEditor(ch)">编辑</button>
            <button class="btn btn-sm btn-danger" @click="removeChannel(ch)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <ModalShell :open="editorOpen" :title="editorId ? '编辑Channel' : '创建Channel'" @close="editorOpen = false">
      <form @submit.prevent="saveEditor">
        <div class="form-group">
          <label>Channel ID（{{ editorId ? '不可修改' : '留空自动生成' }}）</label>
          <input v-model="editor.channel_key" type="text" :disabled="!!editorId" placeholder="留空则自动生成随机ID" />
          <small>8-64位字母, 数字, 连字符或下划线; 公开标识, 可直接嵌在前端URL中</small>
        </div>
        <div class="form-group">
          <label>限流(次/分钟) *</label>
          <input v-model.number="editor.rate_limit" type="number" required min="1" />
        </div>
        <div class="form-group">
          <label>来源绑定（可选）</label>
          <textarea v-model="editor.origins" placeholder="每行一个 origin, 如 https://example.com; 留空不校验来源"></textarea>
        </div>
        <div v-if="editorId" class="form-group">
          <label>状态</label>
          <select v-model="editor.status">
            <option value="active">激活</option>
            <option value="disabled">停用</option>
          </select>
        </div>
        <div v-if="createdId" class="form-group">
          <label>生成的Channel ID</label>
          <div class="token-display">{{ createdId }}</div>
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary" :disabled="!!createdId">
            {{ createdId ? '已创建' : '保存' }}
          </button>
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">
            {{ createdId ? '关闭' : '取消' }}
          </button>
        </div>
      </form>
    </ModalShell>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import type { Channel, ChannelListResponse } from '../../shared/types';
import { adminRequest, formatDate, copyText } from '../../shared/api';
import { toast, confirmModal } from '../ui';
import ModalShell from '../components/ModalShell.vue';

const emit = defineEmits<{ changed: [] }>();

const channels = ref<Channel[]>([]);
const copiedId = ref('');

const editorOpen = ref(false);
const editorId = ref<number | null>(null);
const createdId = ref('');
const editor = reactive({
  channel_key: '',
  rate_limit: 60,
  origins: '',
  status: 'active',
});

onMounted(load);

defineExpose({ reload: load });

async function load() {
  try {
    const data = await adminRequest<ChannelListResponse>('/channels');
    channels.value = data.data || [];
  } catch (error) {
    toast('加载Channels失败: ' + (error as Error).message, 'error');
  }
}

function originsText(ch: Channel): string {
  return (ch.allowed_origins || []).join(', ') || '-';
}

async function copyChannel(ch: Channel) {
  if (await copyText(ch.channel_id)) {
    copiedId.value = ch.channel_id;
    setTimeout(() => (copiedId.value = ''), 1000);
  } else {
    toast('复制失败', 'error');
  }
}

function openEditor(ch: Channel | null) {
  editorId.value = ch?.id ?? null;
  createdId.value = '';
  editor.channel_key = ch?.channel_id ?? '';
  editor.rate_limit = ch?.rate_limit ?? 60;
  editor.origins = (ch?.allowed_origins || []).join('\n');
  editor.status = ch?.status ?? 'active';
  editorOpen.value = true;
}

function parseOrigins(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map(s => s.trim())
    .filter(s => s !== '');
}

async function saveEditor() {
  const payload: Record<string, unknown> = { rate_limit: editor.rate_limit };
  const customID = editor.channel_key.trim();
  if (customID && !editorId.value) payload.channel_id = customID;
  const origins = parseOrigins(editor.origins);
  if (origins.length > 0 || editorId.value) payload.allowed_origins = origins;
  if (editorId.value) payload.status = editor.status;

  try {
    if (editorId.value) {
      await adminRequest(`/channels/${editorId.value}`, { method: 'PUT', body: JSON.stringify(payload) });
      toast('Channel更新成功');
      editorOpen.value = false;
    } else {
      const result = await adminRequest<{ channel_id: string }>('/channels', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      createdId.value = result.channel_id;
      toast('Channel创建成功');
    }
    load();
    emit('changed');
  } catch (error) {
    toast('操作失败: ' + (error as Error).message, 'error');
  }
}

async function removeChannel(ch: Channel) {
  if (!(await confirmModal(`确定要删除 Channel ${ch.channel_id.substring(0, 16)}… 吗？`))) return;
  try {
    await adminRequest(`/channels/${ch.id}`, { method: 'DELETE' });
    toast('Channel删除成功');
    load();
    emit('changed');
  } catch (error) {
    toast('删除失败: ' + (error as Error).message, 'error');
  }
}
</script>
