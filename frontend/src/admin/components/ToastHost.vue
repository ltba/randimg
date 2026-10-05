<template>
  <Teleport to="body">
    <div v-for="t in toasts" :key="t.id" class="alert" :class="`alert-${t.type}`">
      <span class="icon">{{ t.type === 'success' ? '✓' : '✗' }}</span>{{ t.message }}
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useToasts } from '../ui';

const toasts = useToasts();
</script>

<style scoped>
.icon {
  font-weight: 700;
  margin-right: 8px;
}
</style>
