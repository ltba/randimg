// 产物挂载冒烟: happy-dom 环境 import 构建产物, 断言骨架渲染.
// 用法: node scripts/ui-smoke.mjs <产物绝对路径> <admin|gallery|home>
import { Window } from 'happy-dom';

const window = new Window({ url: 'http://localhost/' });
const document = window.document;
document.body.innerHTML = '<div id="app"></div>';

// Vue mount 需要的全局类 (node 环境缺失, happy-dom 提供).
for (const k of ['SVGElement', 'MathMLElement', 'HTMLElement', 'Element', 'Node', 'Document', 'ShadowRoot', 'CustomEvent', 'Event', 'File', 'Blob', 'DOMParser', 'IntersectionObserver']) {
  if (window[k] && !(k in globalThis)) {
    globalThis[k] = window[k];
  }
}

globalThis.window = window;
globalThis.document = document;
Object.defineProperty(globalThis, 'navigator', { value: window.navigator, configurable: true });
globalThis.location = window.location;
globalThis.history = window.history;
globalThis.fetch = () => new Promise(() => {});
const store = new Map();
globalThis.localStorage = {
  getItem: k => (store.has(k) ? store.get(k) : null),
  setItem: (k, v) => store.set(k, String(v)),
  removeItem: k => store.delete(k),
};

const js = process.argv[2];
const page = process.argv[3] || 'admin';
await import(js);

// Vue mount 同步; 等微任务清空.
await new Promise(r => setTimeout(r, 100));

const html = document.body.innerHTML;
const checksByPage = {
  admin: [
    ['sidebar 品牌', html.includes('Rand') && html.includes('Img')],
    ['五个导航项', ['图片管理', '分类管理', 'Channels', '统计数据', '导入'].every(t => html.includes(t))],
    ['统计卡', html.includes('总图片数') && html.includes('今日调用')],
    ['TokenGate (无 token 应显示)', html.includes('请输入管理员 Token')],
    ['背景开关按钮', html.includes('bg-toggle')],
  ],
  gallery: [
    ['页头', html.includes('图片画廊') && html.includes('浏览所有图片')],
    ['筛选下拉', html.includes('所有分类') && html.includes('所有尺寸')],
    ['返回管理链接', html.includes('/admin')],
    ['无限滚动哨兵', html.includes('加载中')],
  ],
  home: [
    ['hero 标题', html.includes('RandImg')],
    ['hero 按钮区', html.includes('管理后台') && html.includes('图片画廊')],
    ['API 文档', html.includes('API 使用文档')],
  ],
};

let fail = 0;
for (const [name, ok] of checksByPage[page] ?? []) {
  console.log(ok ? 'PASS' : 'FAIL', '-', name);
  if (!ok) fail++;
}
if (!checksByPage[page]) {
  console.error('unknown page:', page);
  fail++;
}
process.exit(fail);
