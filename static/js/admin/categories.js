// categories.js — 分类管理域: 表格, 模态框 CRUD; 向其他域提供分类缓存.
import { adminRequest, showAlert, closeModal } from '../core.js';

let categories = [];

export async function getCategories(force = false) {
    if (force || categories.length === 0) {
        categories = await adminRequest('/categories');
    }
    return categories;
}

export function init() {
    document.getElementById('btn-add-category').addEventListener('click', () => showModal());
    document.getElementById('category-form').addEventListener('submit', onSubmit);
}

export async function load() {
    try {
        categories = await adminRequest('/categories', { method: 'GET' });
        const tbody = document.querySelector('#categories-table tbody');
        tbody.innerHTML = '';
        categories.forEach(cat => {
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td>${cat.id}</td>
                <td>${cat.name}</td>
                <td>${cat.slug}</td>
                <td>${cat.description || '-'}</td>
                <td>
                    <button class="btn btn-sm btn-primary" data-action="edit" data-id="${cat.id}">编辑</button>
                    <button class="btn btn-sm btn-danger" data-action="delete" data-id="${cat.id}">删除</button>
                </td>`;
            tbody.appendChild(tr);
        });
    } catch (error) {
        showAlert('加载分类失败: ' + error.message, 'error');
    }
}

function showModal(id = null) {
    if (id) {
        const cat = categories.find(c => c.id === id);
        document.getElementById('category-modal-title').textContent = '编辑分类';
        document.getElementById('category-id').value = cat.id;
        document.getElementById('category-name').value = cat.name;
        document.getElementById('category-slug').value = cat.slug;
        document.getElementById('category-description').value = cat.description || '';
    } else {
        document.getElementById('category-modal-title').textContent = '添加分类';
        document.getElementById('category-form').reset();
        document.getElementById('category-id').value = '';
    }
    document.getElementById('category-modal').classList.add('active');
}

async function onSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('category-id').value;
    const data = {
        name: document.getElementById('category-name').value,
        slug: document.getElementById('category-slug').value,
        description: document.getElementById('category-description').value || null,
    };
    try {
        if (id) {
            await adminRequest(`/categories/${id}`, { method: 'PUT', body: JSON.stringify(data) });
            showAlert('分类更新成功');
        } else {
            await adminRequest('/categories', { method: 'POST', body: JSON.stringify(data) });
            showAlert('分类添加成功');
        }
        closeModal('category-modal');
        load();
    } catch (error) {
        showAlert('操作失败: ' + error.message, 'error');
    }
}

// 表格行内事件委托.
document.addEventListener('click', async (e) => {
    const btn = e.target.closest('#categories-table [data-action]');
    if (!btn) return;
    const id = parseInt(btn.dataset.id);
    if (btn.dataset.action === 'edit') {
        showModal(id);
    } else if (btn.dataset.action === 'delete') {
        if (!confirm('确定要删除这个分类吗？')) return;
        try {
            await adminRequest(`/categories/${id}`, { method: 'DELETE' });
            showAlert('分类删除成功');
            load();
        } catch (error) {
            showAlert('删除失败: ' + error.message, 'error');
        }
    }
});
