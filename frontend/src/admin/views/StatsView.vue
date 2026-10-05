<template>
  <section>
    <div class="page-head"><h2>统计数据</h2></div>
    <p class="section-hint">选择 Channel 查看调用统计</p>
    <div class="form-group" style="max-width: 300px; margin-top: 16px">
      <label>Channel</label>
      <select v-model="channelId" @change="loadLogs">
        <option value="">请选择</option>
        <option v-for="ch in channels" :key="ch.id" :value="ch.id">
          {{ ch.channel_id.substring(0, 24) }}... (ID: {{ ch.id }})
        </option>
      </select>
    </div>
    <table v-if="showTable">
      <thead>
        <tr>
          <th>时间</th>
          <th>Channel ID</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="logs.length === 0">
          <td colspan="2" class="text-center text-2">暂无数据 (统计异步落库, 最多延迟约10秒)</td>
        </tr>
        <tr v-for="(log, i) in logs" :key="i">
          <td>{{ formatDate(log.created_at) }}</td>
          <td>{{ log.channel_id }}</td>
        </tr>
      </tbody>
    </table>
    <div v-if="showTable" class="summary">
      <h3>统计摘要</h3>
      <p>总调用次数: <strong class="num">{{ total }}</strong></p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import type { Channel, ChannelListResponse, StatsResponse } from '../../shared/types';
import { adminRequest, formatDate } from '../../shared/api';
import { toast } from '../ui';

const channels = ref<Channel[]>([]);
const channelId = ref('');
const logs = ref<StatsResponse['logs']>([]);
const total = ref(0);
const showTable = ref(false);

async function reload() {
  try {
    const data = await adminRequest<ChannelListResponse>('/channels');
    channels.value = data.data || [];
  } catch (error) {
    toast('加载Channels失败: ' + (error as Error).message, 'error');
  }
}

onMounted(reload);

defineExpose({ reload });

async function loadLogs() {
  if (!channelId.value) {
    showTable.value = false;
    return;
  }
  try {
    const data = await adminRequest<StatsResponse>(`/stats?channel_id=${channelId.value}`);
    logs.value = data.logs;
    total.value = data.total || data.logs.length;
    showTable.value = true;
  } catch (error) {
    toast('加载统计数据失败: ' + (error as Error).message, 'error');
  }
}
</script>

<style scoped>
.summary {
  margin-top: 20px;
  padding: 15px;
  background: var(--bg);
  border-radius: var(--radius-sm);
}
</style>
