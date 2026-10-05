// images.js — 图片管理域: 表格, 单条 CRUD, 批量选择/更新/删除.
import { adminRequest, showAlert, formatDate, renderPagination, closeModal } from '../core.js';
import { getCategories } from './categories.js';
import { refreshOverview } from './app.js';

let currentPage = 1;
const selectedImages = new Set();

export function init() {
    document.getElementById('btn-add-image').addEventListener('click', () => showModal());
    document.getElementById('image-form').addEventListener('submit', onSubmit);
    document.getElementById('select-all-images').addEventListener('change', toggleSelectAll);
    document.getElementById('btn-batch-update').addEventListener('click', showBatchUpdateModal);
    document.getElementById('btn-batch-delete').addEventListener('click', batchDelete);
    document.getElementById('btn-clear-selection').addEventListener('click', clearSelection);
    document.getElementById('batch-update-form').addEventListener('submit', onBatchUpdate);
    document.getElementById('filter-status').addEventListener('change', () => load(1));
    document.getElementById('filter-fetch').addEventListener('change', () => load(1));

    // URL 列点击复制 (事件委托).
    document.addEventListener('click', (e) => {
        const cell = e.target.closest('#images-table .cell-url');
        if (cell && cell.dataset.url) {
            navigator.clipboard.writeText(cell.dataset.url).then(() => {
                const original = cell.textContent;
                cell.textContent = '已复制!';
                setTimeout(() => { cell.textContent = original; }, 800);
            });
        }
    });

    // 行内编辑/删除与选择框委托.
    document.addEventListener('click', async (e) => {
        const btn = e.target.closest('#images-table [data-action]');
        if (!btn) return;
        const id = parseInt(btn.dataset.id);
        if (btn.dataset.action === 'edit') {
            showModal(id);
        } else if (btn.dataset.action === 'delete') {
            if (!confirm('确定要删除这张图片吗？')) return;
            try {
                await adminRequest(`/images/${id}`, { method: 'DELETE' });
                showAlert('图片删除成功');
                load(currentPage);
            } catch (error) {
                showAlert('删除失败: ' + error.message, 'error');
            }
        }
    });
    document.addEventListener('change', (e) => {
        if (e.target.classList.contains('image-checkbox')) updateSelection();
    });
}

export async function load(page = 1) {
    try {
        const params = new URLSearchParams({ page: String(page), page_size: '20' });
        const status = document.getElementById('filter-status').value;
        const fetchFailed = document.getElementById('filter-fetch').value;
        if (status) params.set('status', status);
        if (fetchFailed) params.set('fetch_failed', fetchFailed);

        const data = await adminRequest(`/images?${params}`);
        const tbody = document.querySelector('#images-table tbody');
        tbody.innerHTML = '';

        data.data.forEach(image => {
            const info = (image.width && image.height)
                ? `${image.width} x ${image.height}`
                : '-';
            const fetchCell = image.fetch_fails >= 3
                ? '<span class="pill pill-warn">失败 ' + image.fetch_fails + '</span>'
                : (info === '-' ? '<span class="pill pill-off">待补全</span>' : '<span class="pill pill-ok">完整</span>');
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td><input type="checkbox" class="image-checkbox" value="${image.id}" ${selectedImages.has(image.id) ? 'checked' : ''}></td>
                <td class="num">${image.id}</td>
                <td><img src="${image.source_url}" class="thumb" loading="lazy" alt=""></td>
                <td class="cell-url" data-url="${image.source_url}" title="${image.source_url} (点击复制)">${image.source_url}</td>
                <td class="num">${info}</td>
                <td>${image.category ? image.category.name : '-'}</td>
                <td>${fetchCell}</td>
                <td><span class="pill ${image.status === 'active' ? 'pill-ok' : 'pill-off'}">${image.status}</span></td>
                <td>
                    <button class="btn btn-sm btn-primary" data-action="edit" data-id="${image.id}">编辑</button>
                    <button class="btn btn-sm btn-danger" data-action="delete" data-id="${image.id}">删除</button>
                </td>`;
            tbody.appendChild(tr);
        });

        renderPagination('images-pagination', data.pagination, load);
        currentPage = page;
        updateSelection();
    } catch (error) {
        showAlert('加载图片失败: ' + error.message, 'error');
    }
}

async function showModal(id = null) {
    const categories = await getCategories();
    const select = document.getElementById('image-category');
    select.innerHTML = categories.map(cat =>
        `<option value="${cat.id}">${cat.name}</option>`).join('');

    if (id) {
        const image = await adminRequest(`/images/${id}`);
        document.getElementById('image-modal-title').textContent = '编辑图片';
        document.getElementById('image-id').value = image.id;
        document.getElementById('image-url').value = image.source_url;
        document.getElementById('image-category').value = image.category_id;
        document.getElementById('image-width').value = image.width || '';
        document.getElementById('image-height').value = image.height || '';
        document.getElementById('image-format').value = image.format || '';
        document.getElementById('image-source').value = image.source || '';
    } else {
        document.getElementById('image-modal-title').textContent = '添加图片';
        document.getElementById('image-form').reset();
        document.getElementById('image-id').value = '';
    }

    document.getElementById('image-modal').classList.add('active');
}

async function onSubmit(e) {
    e.preventDefault();
    const id = document.getElementById('image-id').value;
    const data = {
        source_url: document.getElementById('image-url').value,
        category_id: parseInt(document.getElementById('image-category').value),
        width: document.getElementById('image-width').value ? parseInt(document.getElementById('image-width').value) : null,
        height: document.getElementById('image-height').value ? parseInt(document.getElementById('image-height').value) : null,
        format: document.getElementById('image-format').value || null,
        source: document.getElementById('image-source').value || null,
        auto_fetch: document.getElementById('image-auto-fetch').checked,
    };

    try {
        if (id) {
            await adminRequest(`/images/${id}`, { method: 'PUT', body: JSON.stringify(data) });
            showAlert('图片更新成功');
        } else {
            await adminRequest('/images', { method: 'POST', body: JSON.stringify(data) });
            showAlert('图片添加成功');
        }
        closeModal('image-modal');
        load(currentPage);
        refreshOverview();
    } catch (error) {
        showAlert('操作失败: ' + error.message, 'error');
    }
}

function updateSelection() {
    document.querySelectorAll('.image-checkbox:checked').forEach(cb => selectedImages.add(parseInt(cb.value)));
    document.querySelectorAll('.image-checkbox:not(:checked)').forEach(cb => selectedImages.delete(parseInt(cb.value)));

    const count = selectedImages.size;
    document.getElementById('selected-count').textContent = count;
    document.getElementById('batch-actions').classList.toggle('visible', count > 0);
}

function toggleSelectAll(e) {
    const checked = e.target.checked;
    document.querySelectorAll('.image-checkbox').forEach(cb => {
        cb.checked = checked;
    });
    updateSelection();
}

function clearSelection() {
    document.getElementById('select-all-images').checked = false;
    document.querySelectorAll('.image-checkbox').forEach(cb => {
        cb.checked = false;
    });
    updateSelection();
}

async function showBatchUpdateModal() {
    if (selectedImages.size === 0) {
        showAlert('请先选择要修改的图片', 'error');
        return;
    }
    const categories = await getCategories();
    const select = document.getElementById('batch-category');
    select.innerHTML = '<option value="">不修改</option>' +
        categories.map(cat => `<option value="${cat.id}">${cat.name}</option>`).join('');

    document.getElementById('batch-update-count').textContent = selectedImages.size;
    document.getElementById('batch-update-form').reset();
    document.getElementById('batch-update-modal').classList.add('active');
}

async function onBatchUpdate(e) {
    e.preventDefault();
    const updates = {};
    const categoryId = document.getElementById('batch-category').value;
    const status = document.getElementById('batch-status').value;
    const source = document.getElementById('batch-source').value;

    if (categoryId) updates.category_id = parseInt(categoryId);
    if (status) updates.status = status;
    if (source) updates.source = source;

    if (Object.keys(updates).length === 0) {
        showAlert('请至少选择一项要修改的内容', 'error');
        return;
    }

    try {
        const result = await adminRequest('/images/batch', {
            method: 'PUT',
            body: JSON.stringify({ image_ids: Array.from(selectedImages), updates }),
        });
        showAlert(`批量修改成功！已更新 ${result.updated} 张图片`);
        closeModal('batch-update-modal');
        clearSelection();
        load(currentPage);
    } catch (error) {
        showAlert('批量修改失败: ' + error.message, 'error');
    }
}

async function batchDelete() {
    if (selectedImages.size === 0) {
        showAlert('请先选择要删除的图片', 'error');
        return;
    }
    if (!confirm(`确定要删除选中的 ${selectedImages.size} 张图片吗？`)) return;

    try {
        const result = await adminRequest('/images/batch', {
            method: 'DELETE',
            body: JSON.stringify({ image_ids: Array.from(selectedImages) }),
        });
        showAlert(`批量删除成功！已删除 ${result.deleted} 张图片`);
        clearSelection();
        load(currentPage);
        refreshOverview();
    } catch (error) {
        showAlert('批量删除失败: ' + error.message, 'error');
    }
}
