<template>
  <div class="gallery-container">
    <div class="gallery-header">
      <div>
        <h1>图片画廊</h1>
        <p class="sub">浏览所有图片</p>
      </div>
      <div class="gallery-filters">
        <select v-model="category" @change="reload">
          <option value="">所有分类</option>
          <option v-for="c in categories" :key="c.slug" :value="c.slug">{{ c.name }}</option>
        </select>
        <select v-model="device" @change="reload">
          <option value="">所有尺寸</option>
          <option value="pc">横屏(PC)</option>
          <option value="mobile">竖屏(Mobile)</option>
        </select>
        <a href="/admin" class="btn btn-secondary">返回管理</a>
      </div>
    </div>

    <div class="gallery-grid">
      <div
        v-for="img in images"
        :key="img.id"
        class="gallery-item"
        @click="openLightbox(img)"
      >
        <img :src="img.source_url" :alt="img.category?.name || 'Image'" loading="lazy" />
        <div class="gallery-item-info">
          <span class="category">{{ img.category?.name || '未分类' }}</span>
          <div class="dimensions">
            {{ img.width && img.height ? `${img.width} × ${img.height}` : '尺寸未知' }}
          </div>
        </div>
      </div>
    </div>

    <div ref="sentinel" class="loading-indicator" :class="{ hidden: !loading }">
      <p>加载中...</p>
    </div>
    <p v-if="!loading && images.length === 0" class="loading-indicator">暂无图片</p>

    <Lightbox
      :open="lightboxOpen"
      :images="images"
      :index="lightboxIndex"
      @close="lightboxOpen = false"
      @move="i => (lightboxIndex = i)"
    />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue';
import type { Category, Image, ImageListResponse } from '../shared/types';
import { pubRequest } from '../shared/api';
import Lightbox from './Lightbox.vue';

const categories = ref<Category[]>([]);
const images = ref<Image[]>([]);
const category = ref('');
const device = ref('');

const page = ref(1);
const loading = ref(false);
const hasMore = ref(true);

const lightboxOpen = ref(false);
const lightboxIndex = ref(0);

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function openLightbox(img: Image) {
  lightboxIndex.value = images.value.indexOf(img);
  lightboxOpen.value = true;
}

function reload() {
  images.value = [];
  page.value = 1;
  hasMore.value = true;
  load();
}

async function load() {
  if (loading.value) return;
  loading.value = true;
  try {
    const params = new URLSearchParams({ page: String(page.value), page_size: '20' });
    if (category.value) params.set('category', category.value);
    if (device.value) params.set('device', device.value);
    const data = await pubRequest<ImageListResponse>(`/images?${params}`);
    images.value = images.value.concat(data.data);
    hasMore.value = data.pagination.page < data.pagination.total_page;
    page.value += 1;
  } catch (error) {
    console.error('Failed to load images:', error);
  } finally {
    loading.value = false;
  }
}

function loadMore() {
  if (hasMore.value && !loading.value) load();
}

onMounted(async () => {
  // 哨兵元素驱动无限滚动.
  observer = new IntersectionObserver(
    entries => {
      if (entries.some(e => e.isIntersecting)) loadMore();
    },
    { rootMargin: '200px' },
  );
  await nextTick();
  if (sentinel.value) observer.observe(sentinel.value);

  try {
    categories.value = await pubRequest<Category[]>('/categories');
  } catch (error) {
    console.error('Failed to load categories:', error);
  }
  load();
});

onBeforeUnmount(() => {
  observer?.disconnect();
});
</script>

<style scoped>
.sub {
  color: #666;
  margin-top: 5px;
}
</style>
