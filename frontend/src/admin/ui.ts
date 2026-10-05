// 全局 toast 与确认框状态 (模块级单例, App.vue 挂载容器组件).
import { ref } from 'vue';

export interface ToastItem {
  id: number;
  message: string;
  type: 'success' | 'error';
}

const toasts = ref<ToastItem[]>([]);
let toastSeq = 0;

export function toast(message: string, type: 'success' | 'error' = 'success') {
  const id = ++toastSeq;
  toasts.value.push({ id, message, type });
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id);
  }, 3000);
}

export function useToasts() {
  return toasts;
}

// 确认框: confirmModal(message) -> Promise<boolean>.
let confirmResolve: ((v: boolean) => void) | null = null;
const confirmState = ref<{ message: string; open: boolean }>({ message: '', open: false });

export function confirmModal(message: string): Promise<boolean> {
  return new Promise(resolve => {
    confirmResolve = resolve;
    confirmState.value = { message, open: true };
  });
}

export function resolveConfirm(value: boolean) {
  confirmState.value = { ...confirmState.value, open: false };
  confirmResolve?.(value);
  confirmResolve = null;
}

export function useConfirm() {
  return confirmState;
}
