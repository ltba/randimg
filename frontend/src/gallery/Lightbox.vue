<template>
  <Teleport to="body">
    <div v-if="open" class="lightbox" @click.self="$emit('close')">
      <button class="lightbox-close" type="button" @click="$emit('close')">&times;</button>
      <button class="lightbox-nav lightbox-prev" type="button" @click="step(-1)">‹</button>
      <img :src="current?.source_url" alt="" @load="loading = false" @error="loading = false" />
      <button class="lightbox-nav lightbox-next" type="button" @click="step(1)">›</button>
      <div v-if="loading" class="lightbox-loading">加载中...</div>
      <div class="lightbox-caption">
        #{{ current?.id }} · {{ current?.category?.name || '未分类' }}
        <template v-if="current?.width && current?.height"> · {{ current.width }} × {{ current.height }}</template>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue';
import type { Image } from '../shared/types';

const props = defineProps<{ open: boolean; images: Image[]; index: number }>();
const emit = defineEmits<{ close: []; move: [index: number] }>();

const loading = ref(false);
const current = computed(() => props.images[props.index] ?? null);

watch(
  () => [props.open, props.index] as const,
  () => {
    loading.value = true;
  },
);

function step(dir: number) {
  const n = props.images.length;
  if (n === 0) return;
  emit('move', (props.index + dir + n) % n);
}

function onKey(e: KeyboardEvent) {
  if (!props.open) return;
  if (e.key === 'Escape') emit('close');
  else if (e.key === 'ArrowLeft') step(-1);
  else if (e.key === 'ArrowRight') step(1);
}

onMounted(() => document.addEventListener('keydown', onKey));
onUnmounted(() => document.removeEventListener('keydown', onKey));
</script>

<style scoped>
.lightbox-loading {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #e2e8f0;
  font-size: 14px;
}

.lightbox-caption {
  position: absolute;
  bottom: 18px;
  left: 0;
  right: 0;
  text-align: center;
  color: #e2e8f0;
  font-size: 13px;
  text-shadow: 0 1px 4px rgba(0, 0, 0, 0.8);
}
</style>
