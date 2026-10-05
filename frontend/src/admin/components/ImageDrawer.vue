<template>
  <Teleport to="body">
    <div v-if="image" class="drawer-mask" @click.self="$emit('close')">
      <aside class="drawer">
        <header>
          <h2>图片详情 #{{ image.id }}</h2>
          <button class="close" type="button" @click="$emit('close')">&times;</button>
        </header>

        <div class="drawer-body">
          <a :href="image.source_url" target="_blank" rel="noopener">
            <img class="hero" :src="image.source_url" :alt="`image ${image.id}`" loading="lazy" />
          </a>

          <dl class="meta">
            <dt>尺寸</dt>
            <dd class="num">{{ image.width && image.height ? `${image.width} × ${image.height}` : '待补全' }}</dd>
            <dt>格式</dt>
            <dd>{{ image.format || '-' }}</dd>
            <dt>分类</dt>
            <dd>{{ image.category?.name || '-' }}</dd>
            <dt>来源</dt>
            <dd>{{ image.source || '-' }}</dd>
            <dt>状态</dt>
            <dd>
              <span class="pill" :class="image.status === 'active' ? 'pill-ok' : 'pill-off'">{{ image.status }}</span>
            </dd>
            <dt>补全</dt>
            <dd>
              <span v-if="image.fetch_fails >= 3" class="pill pill-warn">失败 {{ image.fetch_fails }} 次</span>
              <span v-else-if="!image.width || !image.height" class="pill pill-off">待补全</span>
              <span v-else class="pill pill-ok">完整</span>
            </dd>
            <dt>创建时间</dt>
            <dd>{{ formatDate(image.created_at) }}</dd>
            <dt>更新时间</dt>
            <dd>{{ formatDate(image.updated_at) }}</dd>
          </dl>

          <div class="url-rows">
            <div class="url-row" @click="copy(image.source_url)">
              <span class="label">源 URL</span>
              <span class="value" :title="image.source_url">{{ image.source_url }}</span>
              <span class="copy-hint">{{ copied === 'source' ? '已复制!' : '复制' }}</span>
            </div>
            <div class="url-row" @click="copy(`/api/proxy/${image.id}`, 'proxy')">
              <span class="label">代理 URL</span>
              <span class="value">/api/proxy/{{ image.id }}</span>
              <span class="copy-hint">{{ copied === 'proxy' ? '已复制!' : '复制' }}</span>
            </div>
          </div>
        </div>

        <footer>
          <button class="btn btn-primary" @click="$emit('edit', image)">编辑</button>
          <button class="btn btn-danger" @click="$emit('remove', image)">删除</button>
        </footer>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Image } from '../../shared/types';
import { formatDate, copyText } from '../../shared/api';
import { toast } from '../ui';

defineProps<{ image: Image | null }>();
defineEmits<{ close: []; edit: [image: Image]; remove: [image: Image] }>();

const copied = ref<'source' | 'proxy' | null>(null);

async function copy(text: string, which: 'source' | 'proxy' = 'source') {
  if (await copyText(text)) {
    copied.value = which;
    setTimeout(() => (copied.value = null), 1000);
  } else {
    toast('复制失败', 'error');
  }
}
</script>

<style scoped>
.drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 1500;
  background: rgba(15, 23, 42, 0.35);
}

.drawer {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 440px;
  max-width: 92vw;
  background: var(--surface);
  box-shadow: var(--shadow-md);
  display: flex;
  flex-direction: column;
  animation: slide-in 0.2s ease;
}

@keyframes slide-in {
  from {
    transform: translateX(60px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-2);
  line-height: 1;
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.hero {
  width: 100%;
  max-height: 300px;
  object-fit: contain;
  border-radius: var(--radius-sm);
  background: var(--bg);
}

.meta {
  display: grid;
  grid-template-columns: 84px 1fr;
  gap: 8px 12px;
  margin-top: 18px;
  font-size: 13px;
}

.meta dt {
  color: var(--text-2);
}

.meta dd {
  word-break: break-all;
}

.url-rows {
  margin-top: 18px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.url-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
}

.url-row:hover {
  border-color: var(--primary);
}

.label {
  color: var(--text-2);
  white-space: nowrap;
}

.value {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, Menlo, Consolas, monospace;
}

.copy-hint {
  color: var(--primary);
  white-space: nowrap;
}

footer {
  display: flex;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
}
</style>
