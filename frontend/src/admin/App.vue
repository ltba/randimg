<template>
  <div class="layout">
    <div class="bg-layer" :style="bgStyle"></div>
    <div v-if="bgEnabled" class="bg-overlay"></div>

    <button
      v-if="bgEnabled"
      class="bg-btn bg-refresh"
      type="button"
      title="换一张背景"
      @click="refreshBg"
    >↻</button>
    <button
      class="bg-btn bg-toggle"
      :class="{ off: !bgEnabled }"
      type="button"
      :title="bgEnabled ? '关闭背景图' : '开启背景图'"
      @click="toggleBg"
    >{{ bgEnabled ? '✕' : '🖼' }}</button>

    <aside class="sidebar">
      <div class="sidebar-brand">Rand<span>Img</span></div>
      <nav class="sidebar-nav">
        <button
          v-for="s in sections"
          :key="s.key"
          :class="{ active: current === s.key }"
          @click="switchTo(s.key)"
        >{{ s.label }}</button>
      </nav>
      <div class="sidebar-foot">
        <a href="/">首页</a>
        <a href="/gallery">画廊</a>
      </div>
    </aside>

    <main class="main">
      <div class="stats">
        <div class="stat-card">
          <h3>总图片数</h3>
          <p class="num">{{ overview?.total_images ?? '-' }}</p>
        </div>
        <div class="stat-card">
          <h3>Channels</h3>
          <p class="num">{{ overview?.active_channels ?? '-' }}</p>
        </div>
        <div class="stat-card">
          <h3>今日调用</h3>
          <p class="num">{{ overview?.today_calls ?? '-' }}</p>
          <p v-if="overview" class="stat-sub">匿名 {{ overview.today_anon_calls }}</p>
        </div>
        <div class="stat-card">
          <h3>总调用次数</h3>
          <p class="num">{{ overview?.total_calls ?? '-' }}</p>
          <p v-if="overview" class="stat-sub">匿名 {{ overview.total_anon_calls }}</p>
        </div>
      </div>

      <div class="content">
        <ImagesView
          v-show="current === 'images'"
          ref="imagesRef"
          :categories="categories"
          @changed="refreshAll"
        />
        <CategoriesView v-show="current === 'categories'" ref="categoriesRef" @updated="loadCategories" />
        <ChannelsView v-show="current === 'channels'" ref="channelsRef" @changed="refreshAll" />
        <StatsView v-show="current === 'stats'" ref="statsRef" />
        <ImportView v-show="current === 'import'" :categories="categories" @changed="refreshAll" />
      </div>
    </main>

    <TokenGate />
    <ToastHost />
    <ConfirmHost />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import type { Category, OverviewStats } from '../shared/types';
import { adminRequest } from '../shared/api';
import ImagesView from './views/ImagesView.vue';
import CategoriesView from './views/CategoriesView.vue';
import ChannelsView from './views/ChannelsView.vue';
import StatsView from './views/StatsView.vue';
import ImportView from './views/ImportView.vue';
import TokenGate from './components/TokenGate.vue';
import ToastHost from './components/ToastHost.vue';
import ConfirmHost from './components/ConfirmHost.vue';

const imagesRef = ref<InstanceType<typeof ImagesView> | null>(null);
const categoriesRef = ref<InstanceType<typeof CategoriesView> | null>(null);
const channelsRef = ref<InstanceType<typeof ChannelsView> | null>(null);
const statsRef = ref<InstanceType<typeof StatsView> | null>(null);

const sections = [
  { key: 'images', label: '图片管理' },
  { key: 'categories', label: '分类管理' },
  { key: 'channels', label: 'Channels' },
  { key: 'stats', label: '统计数据' },
  { key: 'import', label: '导入' },
] as const;

type SectionKey = (typeof sections)[number]['key'];

const current = ref<SectionKey>('images');
const categories = ref<Category[]>([]);
const overview = ref<OverviewStats | null>(null);

// section 记忆在 hash: 刷新保持当前视图.
const fromHash = location.hash.replace('#', '') as SectionKey;
if (sections.some(s => s.key === fromHash)) current.value = fromHash;
watch(current, v => {
  history.replaceState(null, '', `#${v}`);
  // 切换视图时重拉该域数据: 导入后切回图片管理即可见最新状态 (含补全进度).
  const refs: Partial<Record<SectionKey, { reload?: () => void } | null>> = {
    images: imagesRef.value,
    categories: categoriesRef.value,
    channels: channelsRef.value,
    stats: statsRef.value,
  };
  // import 视图数据由 App 的 categories 驱动, 无需重拉.
  refs[v]?.reload?.();
});

// 随机图背景: 开关记 localStorage, 换一张换 seed.
const bgEnabled = ref(localStorage.getItem('admin_bg') !== 'off');
const bgSeed = ref(Date.now());

const bgStyle = computed(() => ({
  // 背景匿名调用: 不计量 (统计仅计 Channel), 受匿名桶兜底限流.
  backgroundImage: bgEnabled.value
    ? `url('/api/random?category=acg&t=${bgSeed.value}')`
    : 'none',
}));

function refreshBg() {
  bgSeed.value = Date.now();
}

function toggleBg() {
  bgEnabled.value = !bgEnabled.value;
  localStorage.setItem('admin_bg', bgEnabled.value ? 'on' : 'off');
}

watch(bgEnabled, v => {
  document.body.classList.toggle('has-bg', v);
}, { immediate: true });

function switchTo(key: SectionKey) {
  current.value = key;
}

async function loadOverview() {
  try {
    overview.value = await adminRequest<OverviewStats>('/stats/overview');
  } catch {
    // 概览失败静默: TokenGate 会处理 401.
  }
}

async function loadCategories() {
  try {
    categories.value = await adminRequest<Category[]>('/categories');
  } catch {
    // 同上.
  }
}

function refreshAll() {
  loadOverview();
  loadCategories();
}

onMounted(refreshAll);
</script>

<style scoped>
.bg-layer {
  position: fixed;
  inset: 0;
  z-index: -2;
  background-size: cover;
  background-position: center;
}

.bg-overlay {
  position: fixed;
  inset: 0;
  z-index: -1;
  background: rgba(241, 245, 249, 0.82);
}

.bg-btn {
  position: fixed;
  z-index: 110;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text-2);
  cursor: pointer;
  font-size: 15px;
  line-height: 1;
  box-shadow: var(--shadow-sm);
}

.bg-btn:hover {
  color: var(--text);
}

.bg-refresh {
  top: 14px;
  right: 58px;
}

.bg-toggle {
  top: 14px;
  right: 14px;
}

.bg-toggle.off {
  opacity: 0.6;
}
</style>
