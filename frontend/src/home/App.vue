<template>
  <div>
    <div class="background" id="background" :style="{ backgroundImage: `url('/api/random?category=acg&t=${bgSeed}')` }"></div>
    <div class="background-overlay"></div>

    <div class="container">
      <h1>🎨 RandImg</h1>
      <p class="subtitle">强大的随机图片API服务 - 支持多图源、缓存、限流、代理</p>

      <div class="features">
        <div class="feature">
          <div class="feature-icon">🚀</div>
          <div class="feature-title">高性能</div>
          <div class="feature-desc">内存限流+异步统计</div>
        </div>
        <div class="feature">
          <div class="feature-icon">🔒</div>
          <div class="feature-title">安全可靠</div>
          <div class="feature-desc">API Key认证</div>
        </div>
        <div class="feature">
          <div class="feature-icon">🎯</div>
          <div class="feature-title">灵活配置</div>
          <div class="feature-desc">多种返回格式</div>
        </div>
        <div class="feature">
          <div class="feature-icon">📊</div>
          <div class="feature-title">数据统计</div>
          <div class="feature-desc">完整的使用分析</div>
        </div>
      </div>

      <div class="api-docs">
        <h2>📖 API 使用文档</h2>

        <div class="api-section">
          <h3>1. 随机图片接口</h3>
          <div class="api-endpoint"><code>GET /api/random</code></div>
          <div class="api-params">
            <h4>参数说明：</h4>
            <table>
              <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
              <tr><td><code>category</code></td><td>string</td><td>否</td><td>分类slug（如：acg、wallpaper）</td></tr>
              <tr><td><code>device</code></td><td>string</td><td>否</td><td>设备类型：<code>pc</code>（横屏）或 <code>mobile</code>（竖屏）<br><strong>不指定时自动从User-Agent识别，默认pc</strong></td></tr>
              <tr><td><code>min_width</code> / <code>max_width</code> / <code>min_height</code> / <code>max_height</code></td><td>int</td><td>否</td><td>尺寸范围过滤：正整数， 可任意组合， 与 device 叠加； 如 <code>?min_width=1920</code></td></tr>
              <tr><td><code>output</code></td><td>string</td><td>否</td><td>响应模式：<code>redirect</code>（默认，302重定向）、<code>json</code>（JSON数据）、<code>proxy</code>（代理图片）</td></tr>
              <tr><td><code>compress</code></td><td>boolean</td><td>否</td><td>是否压缩图片（仅proxy模式有效）：<code>true</code> 或 <code>false</code>（默认）</td></tr>
              <tr><td><code>channel_id</code></td><td>string</td><td>否</td><td>接入渠道标识：用于计量与独立限流（可在管理后台创建；不携带则计入匿名统计（不计入总调用），受全局兜底限流）</td></tr>
            </table>
          </div>
          <div class="api-examples">
            <h4>使用示例：</h4>
            <pre># 最简单的用法（自动识别设备, 同源相对路径）
&lt;img src="{{ origin }}/api/random"&gt;

# 指定分类和设备
&lt;img src="{{ origin }}/api/random?category=acg&amp;device=mobile"&gt;

# 获取JSON数据
curl "{{ origin }}/api/random?output=json"

# 携带Channel（计量与独立限流）
curl "{{ origin }}/api/random?channel_id=YOUR_CHANNEL&amp;output=json"</pre>
          </div>
        </div>

        <div class="api-section">
          <h3>2. 图片代理接口</h3>
          <div class="api-endpoint"><code>GET /api/proxy/:id</code></div>
          <div class="api-params">
            <h4>参数说明：</h4>
            <table>
              <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
              <tr><td><code>id</code></td><td>number</td><td>是</td><td>图片ID（路径参数）</td></tr>
              <tr><td><code>compress</code></td><td>boolean</td><td>否</td><td>是否压缩：<code>true</code> 或 <code>false</code>（默认）</td></tr>
              <tr><td><code>format</code></td><td>string</td><td>否</td><td>目标格式：<code>jpeg</code>、<code>png</code></td></tr>
            </table>
          </div>
          <div class="api-examples">
            <h4>使用示例：</h4>
            <pre># 代理原图
&lt;img src="{{ origin }}/api/proxy/123"&gt;

# 压缩并转换为JPEG
&lt;img src="{{ origin }}/api/proxy/123?compress=true&amp;format=jpeg"&gt;</pre>
          </div>
        </div>

        <div class="api-section">
          <h3>3. 图片列表接口</h3>
          <div class="api-endpoint"><code>GET /api/images</code></div>
          <div class="api-params">
            <h4>参数说明：</h4>
            <table>
              <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
              <tr><td><code>page</code></td><td>number</td><td>否</td><td>页码（默认1）</td></tr>
              <tr><td><code>page_size</code></td><td>number</td><td>否</td><td>每页数量（默认20，最大100）</td></tr>
              <tr><td><code>category</code></td><td>string</td><td>否</td><td>分类slug</td></tr>
              <tr><td><code>device</code></td><td>string</td><td>否</td><td>设备类型：<code>pc</code> 或 <code>mobile</code></td></tr>
            </table>
          </div>
        </div>

        <div class="api-section">
          <h3>4. 分类列表接口</h3>
          <div class="api-endpoint"><code>GET /api/categories</code></div>
          <div class="api-examples">
            <h4>使用示例：</h4>
            <pre>curl "{{ origin }}/api/categories"</pre>
          </div>
        </div>

        <div class="api-section">
          <h3>🔑 接入说明</h3>
          <ul>
            <li><strong>公开访问</strong>：公开接口无需任何凭证，跨域默认放行（CORS 全局开放）</li>
            <li><strong>Channel</strong>：可携带 <code>channel_id</code> 参数，用于流量计量与独立限流；它不是秘密，可直接嵌在前端URL中</li>
            <li><strong>匿名访问</strong>：不携带 channel_id 的请求计入匿名统计（不计入总调用），受全局兜底限流保护</li>
            <li><strong>管理API</strong>：需要在请求头添加 <code>Authorization: Bearer ADMIN_TOKEN</code></li>
          </ul>
        </div>

        <div class="api-section">
          <h3>⚡ 限流规则</h3>
          <ul>
            <li>携带 channel_id：按 Channel 配置的限流规则（滑动窗口，默认 60 次/分钟）</li>
            <li>匿名请求：全局兑底限流（默认 300 次/分钟，可用 ANON_RATE_LIMIT 配置）</li>
            <li>超限返回 429，响应携带 X-RateLimit-Limit 与 X-RateLimit-Remaining</li>
          </ul>
        </div>
      </div>

      <div class="api-demo">
        <h3>🚀 快速开始</h3>
        <div># 获取随机图片（JSON格式）</div>
        <div>curl "<code>{{ origin }}/api/random?output=json</code>"</div>
        <br />
        <div># 获取随机图片（302重定向）</div>
        <div>curl -L "<code>{{ origin }}/api/random</code>"</div>
        <br />
        <div># 按分类获取</div>
        <div>curl "<code>{{ origin }}/api/random?category=acg</code>"</div>
      </div>

      <div class="stats">
        <div class="stat">
          <div class="stat-value num">{{ stats?.total_images ?? '-' }}</div>
          <div class="stat-label">总图片数</div>
        </div>
        <div class="stat">
          <div class="stat-value num">{{ stats?.total_calls ?? '-' }}</div>
          <div class="stat-label">总调用次数</div>
        </div>
        <div class="stat">
          <div class="stat-value num">{{ stats?.today_calls ?? '-' }}</div>
          <div class="stat-label">今日调用</div>
        </div>
      </div>
    </div>

    <div class="fab-group">
      <a href="/admin" class="fab" title="管理后台" aria-label="管理后台">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
          <circle cx="12" cy="12" r="3" />
        </svg>
      </a>
      <a href="/gallery" class="fab" title="图片画廊" aria-label="图片画廊">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
          <circle cx="9" cy="9" r="2" />
          <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
        </svg>
      </a>
      <a href="https://github.com/ltba/randimg" class="fab" title="GitHub" aria-label="GitHub" target="_blank" rel="noopener">
        <svg viewBox="0 0 24 24" fill="currentColor" stroke="none">
          <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
        </svg>
      </a>
      <button class="fab" type="button" title="换一张背景" aria-label="换一张背景" @click="refreshBg">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2" />
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { pubRequest } from '../shared/api';

interface HomeStats {
  total_images: number;
  total_calls: number;
  today_calls: number;
}

const stats = ref<HomeStats | null>(null);
// 文档示例动态展示当前访问域名, 复制即用.
const origin = location.origin;
const bgSeed = ref(Date.now());

function refreshBg() {
  const bg = document.getElementById('background');
  if (!bg) return;
  bg.style.opacity = '0';
  setTimeout(() => {
    bgSeed.value = Date.now();
    bg.style.transition = 'opacity 0.5s';
    bg.style.opacity = '1';
  }, 500);
}

onMounted(async () => {
  try {
    stats.value = await pubRequest<HomeStats>('/stats');
  } catch (error) {
    console.error('Failed to load stats:', error);
  }
});
</script>

<style scoped>
</style>
