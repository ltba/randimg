<template>
  <div v-if="token === null || token === ''" class="token-gate">
    <div class="token-card">
      <h2>RandImg 管理后台</h2>
      <p class="hint">请输入管理员 Token 继续</p>
      <form @submit.prevent="submit">
        <input
          ref="inputEl"
          v-model="value"
          type="password"
          placeholder="ADMIN_TOKEN"
          autocomplete="current-password"
          autofocus
        />
        <button type="submit" class="btn btn-primary" :disabled="!value.trim()">进入</button>
      </form>
      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import { useAdminToken, setAdminToken } from '../../shared/api';

const token = useAdminToken();
const value = ref('');
const error = ref('');
const inputEl = ref<HTMLInputElement | null>(null);

watch(
  () => token.value,
  async t => {
    if (t === null || t === '') {
      value.value = '';
      error.value = 'Token 无效或已过期, 请重新输入';
      await nextTick();
      inputEl.value?.focus();
    }
  },
);

function submit() {
  const v = value.value.trim();
  if (!v) return;
  setAdminToken(v);
  error.value = '';
}
</script>

<style scoped>
.token-gate {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(15, 23, 42, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.token-card {
  background: var(--surface);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  padding: 32px;
  width: 360px;
  max-width: 90vw;
  animation: pop 0.18s ease;
}

.token-card h2 {
  margin-bottom: 6px;
}

.hint {
  color: var(--text-2);
  font-size: 13px;
  margin-bottom: 18px;
}

form {
  display: flex;
  gap: 10px;
}

input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 14px;
}

input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-soft);
}

.error {
  color: #b91c1c;
  font-size: 13px;
  margin-top: 12px;
}
</style>
