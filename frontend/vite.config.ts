import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { resolve } from 'node:path';

// 三页多入口 (MPA): /admin, /, /gallery 由 Go 组合根分别服务对应产物.
export default defineConfig({
  plugins: [vue()],
  build: {
    rollupOptions: {
      input: {
        admin: resolve(__dirname, 'admin.html'),
        home: resolve(__dirname, 'home.html'),
        gallery: resolve(__dirname, 'gallery.html'),
      },
    },
  },
  server: {
    proxy: {
      // dev 时代理到本地 Go 后端.
      '/api': 'http://localhost:8080',
    },
  },
});
