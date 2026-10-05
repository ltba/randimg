// core.js — 唯一请求层与共享 UI 工具.
// 管理端请求自动附带 Bearer token (401 时清除并要求重输); 公开请求无凭证.

const ADMIN_BASE = '/api/admin';
const PUB_BASE = '/api';

export function getAdminToken() {
    let token = localStorage.getItem('admin_token');
    if (!token) {
        token = prompt('请输入管理员Token:');
        if (token) {
            localStorage.setItem('admin_token', token.trim());
        }
    }
    return token;
}

export function clearAdminToken() {
    localStorage.removeItem('admin_token');
}

export async function adminRequest(url, options = {}) {
    const token = getAdminToken();
    if (!token) {
        throw new Error('未设置管理员Token');
    }

    const response = await fetch(ADMIN_BASE + url, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            ...options.headers,
        },
    });

    if (!response.ok) {
        if (response.status === 401) {
            clearAdminToken();
            throw new Error('Token无效，请刷新页面重新输入');
        }
        const err = await response.json().catch(() => ({}));
        throw new Error(err.error || `HTTP ${response.status}`);
    }

    return await response.json();
}

export async function pubRequest(url, options = {}) {
    const response = await fetch(PUB_BASE + url, options);
    if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.error || `HTTP ${response.status}`);
    }
    return await response.json();
}

export function showAlert(message, type = 'success') {
    const container = document.getElementById('alert-container') || document.body;
    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;
    container.appendChild(alert);
    setTimeout(() => alert.remove(), 3000);
}

export function formatDate(dateString) {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleString('zh-CN');
}

export function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

// 统一模态框关闭: 任意 [data-dismiss] 元素关闭其所属 modal.
document.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-dismiss]');
    if (btn) closeModal(btn.dataset.dismiss);
});

// renderPagination 分页组件: 当前页前后各 2 页, 首尾页省略号.
export function renderPagination(containerId, pagination, loadFunc) {
    const container = document.getElementById(containerId);
    container.innerHTML = '';

    const totalPages = pagination.total_page;
    const page = pagination.page;
    if (totalPages <= 1) return;

    const append = (text, target, isCurrent) => {
        const el = document.createElement(text === '...' ? 'span' : 'button');
        el.textContent = text;
        if (text !== '...') {
            if (isCurrent) el.className = 'active';
            el.addEventListener('click', () => loadFunc(target));
        }
        container.appendChild(el);
    };

    if (page > 1) append('«', page - 1);

    const start = Math.max(1, page - 2);
    const end = Math.min(totalPages, page + 2);

    if (start > 1) {
        append('1', 1);
        if (start > 2) append('...', 0);
    }
    for (let i = start; i <= end; i++) append(String(i), i, i === page);
    if (end < totalPages) {
        if (end < totalPages - 1) append('...', 0);
        append(String(totalPages), totalPages);
    }

    if (page < totalPages) append('»', page + 1);
}
