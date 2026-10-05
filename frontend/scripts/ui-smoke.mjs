// admin 产物挂载冒烟: happy-dom 环境 import 构建产物, 断言骨架渲染.
import { Window } from 'happy-dom';

const window = new Window({ url: 'http://localhost/admin' });
const document = window.document;
document.body.innerHTML = '<div id="app"></div>';

// Vue mount 需要 SVGElement/MathMLElement 全局类.
for (const k of ['SVGElement', 'MathMLElement', 'HTMLElement', 'Element', 'Node', 'Document', 'ShadowRoot', 'CustomEvent', 'Event', 'File', 'Blob', 'DOMParser']) {
  if (window[k] && !(k in globalThis)) {
    globalThis[k] = window[k];
  }
}

// stub: fetch 返回 pending, localStorage 内存实现.
globalThis.window = window;
globalThis.document = document;
Object.defineProperty(globalThis, "navigator", { value: window.navigator, configurable: true });
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
await import(js);

// Vue mount 是同步的 (无 Suspense); 等微任务清空.
await new Promise(r => setTimeout(r, 100));

const html = document.body.innerHTML;
const checks = [
  ['sidebar 品牌', html.includes('Rand') && html.includes('Img')],
  ['五个导航项', ['图片管理', '分类管理', 'Channels', '统计数据', '导入'].every(t => html.includes(t))],
  ['统计卡', html.includes('总图片数') && html.includes('今日调用')],
  ['TokenGate (无 token 应显示)', html.includes('请输入管理员 Token')],
  ['背景开关按钮', html.includes('bg-toggle')],
];
let fail = 0;
for (const [name, ok] of checks) {
  console.log(ok ? 'PASS' : 'FAIL', '-', name);
  if (!ok) fail++;
}
process.exit(fail);
