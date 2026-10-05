// 唯一请求层: token 管理 (ref 响应式, TokenGate 消费), 管理端/公开端请求.
import { ref } from 'vue';

const ADMIN_BASE = '/api/admin';
const PUB_BASE = '/api';

const adminToken = ref<string | null>(localStorage.getItem('admin_token'));

export function useAdminToken() {
  return adminToken;
}

export function setAdminToken(token: string) {
  adminToken.value = token;
  localStorage.setItem('admin_token', token);
}

export function clearAdminToken() {
  adminToken.value = null;
  localStorage.removeItem('admin_token');
}

export async function adminRequest<T>(url: string, options: RequestInit = {}): Promise<T> {
  if (!adminToken.value) {
    throw new Error('未设置管理员Token');
  }
  const response = await fetch(ADMIN_BASE + url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${adminToken.value}`,
      ...options.headers,
    },
  });
  if (!response.ok) {
    if (response.status === 401) {
      clearAdminToken();
    }
    const err = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error || `HTTP ${response.status}`);
  }
  return (await response.json()) as T;
}

export async function pubRequest<T>(url: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(PUB_BASE + url, options);
  if (!response.ok) {
    const err = (await response.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error || `HTTP ${response.status}`);
  }
  return (await response.json()) as T;
}

export function formatDate(dateString: string | null | undefined): string {
  if (!dateString) return '-';
  return new Date(dateString).toLocaleString('zh-CN');
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
