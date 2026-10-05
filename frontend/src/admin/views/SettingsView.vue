<template>
  <section>
    <div class="page-head"><h2>设置</h2></div>

    <form class="form-group" @submit.prevent="saveAnonLimit">
      <label for="anon-limit-input">匿名限流 (次/分钟)</label>
      <div class="setting-row">
        <input id="anon-limit-input" v-model.number="anonLimit" type="number" min="1" max="100000" required />
        <button type="submit" class="btn btn-sm btn-primary" :disabled="savingAnon">
          {{ savingAnon ? '保存中...' : '保存' }}
        </button>
      </div>
      <small>全部匿名调用共享的单桶滑动窗口阈值; 保存后立即生效</small>
    </form>

    <div class="form-group">
      <label>背景图</label>
      <label class="setting-check">
        <input type="checkbox" :checked="bgEnabled" @change="emit('toggle-bg')" />
        启用随机背景图
      </label>
      <small>管理页背景随机取图; 也可用右上角按钮快速开关</small>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminRequest } from '../../shared/api';
import { toast } from '../ui';

defineProps<{ bgEnabled: boolean }>();

const emit = defineEmits<{ 'toggle-bg': [] }>();

const anonLimit = ref(0);
const savingAnon = ref(false);

onMounted(loadAnonLimit);

defineExpose({ reload: loadAnonLimit });

async function loadAnonLimit() {
  try {
    const data = await adminRequest<{ anon_rate_limit: number }>('/settings/anon');
    anonLimit.value = data.anon_rate_limit;
  } catch (error) {
    toast('加载匿名限流失败: ' + (error as Error).message, 'error');
  }
}

async function saveAnonLimit() {
  if (!anonLimit.value || anonLimit.value < 1) return;
  savingAnon.value = true;
  try {
    await adminRequest('/settings/anon', { method: 'PUT', body: JSON.stringify({ anon_rate_limit: anonLimit.value }) });
    toast('匿名限流已更新');
  } catch (error) {
    toast('保存失败: ' + (error as Error).message, 'error');
  } finally {
    savingAnon.value = false;
  }
}
</script>
