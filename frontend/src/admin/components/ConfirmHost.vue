<template>
  <Teleport to="body">
    <div v-if="state.open" class="modal active">
      <div class="modal-content confirm-card">
        <h2>确认操作</h2>
        <p class="message">{{ state.message }}</p>
        <div class="form-actions">
          <button class="btn btn-danger" @click="resolve(true)">确认</button>
          <button class="btn btn-secondary" @click="resolve(false)">取消</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useConfirm, resolveConfirm } from '../ui';

const state = useConfirm();
function resolve(v: boolean) {
  resolveConfirm(v);
}
</script>

<style scoped>
.confirm-card {
  max-width: 400px;
}

.message {
  color: var(--text-2);
  line-height: 1.6;
  white-space: pre-line;
}
</style>
