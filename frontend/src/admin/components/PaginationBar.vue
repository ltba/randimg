<template>
  <div v-if="totalPage > 1 || pageSizeOptions.length" class="pagination">
    <button v-if="page > 1" @click="go(page - 1)">«</button>
    <button v-if="start > 1" @click="go(1)">1</button>
    <span v-if="start > 2">...</span>
    <button v-for="p in mid" :key="p" :class="{ active: p === page }" @click="go(p)">{{ p }}</button>
    <span v-if="end < totalPage - 1">...</span>
    <button v-if="end < totalPage" @click="go(totalPage)">{{ totalPage }}</button>
    <button v-if="page < totalPage" @click="go(page + 1)">»</button>

    <span class="sep">|</span>
    <select :value="pageSize" class="size-select" @change="onSize">
      <option v-for="s in pageSizeOptions" :key="s" :value="s">{{ s }} 条/页</option>
    </select>
    <span class="jump">
      跳至
      <input
        :value="jumpValue"
        type="number"
        min="1"
        :max="totalPage"
        @change="onJump($event)"
      />
      页
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    page: number;
    pageSize: number;
    totalPage: number;
    pageSizeOptions?: number[];
  }>(),
  { pageSizeOptions: () => [20, 50, 100] },
);

const emit = defineEmits<{ 'go': [page: number]; 'size': [size: number] }>();

const jumpValue = ref('');
watch(
  () => props.page,
  () => {
    jumpValue.value = '';
  },
);

const start = computed(() => Math.max(1, props.page - 2));
const end = computed(() => Math.min(props.totalPage, props.page + 2));
const mid = computed(() => {
  const out: number[] = [];
  for (let i = start.value; i <= end.value; i++) out.push(i);
  return out;
});

function go(p: number) {
  if (p >= 1 && p <= props.totalPage && p !== props.page) emit('go', p);
}

function onSize(e: Event) {
  emit('size', parseInt((e.target as HTMLSelectElement).value, 10));
}

function onJump(e: Event) {
  const v = parseInt((e.target as HTMLInputElement).value, 10);
  if (!Number.isNaN(v)) go(v);
  jumpValue.value = '';
}
</script>

<style scoped>
.sep {
  padding: 6px 2px;
  color: var(--border);
}

.size-select {
  padding: 5px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface);
  font-size: 13px;
}

.jump {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-2);
  font-size: 13px;
}

.jump input {
  width: 56px;
  padding: 5px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
}
</style>
