// channels.js — Channel 管理域: 表格, 创建/编辑/删除, channel_id 点击复制.
import { adminRequest, showAlert, formatDate, closeModal } from '../core.js';
import { refreshOverview } from './app.js';

let channels = [];

export function init() {
    document.getElementById('btn-add-channel').addEventListener('click', () => showModal());
    document.getElementById('channel-form').addEventListener('submit', onSubmit);

    document.addEventListener('click', async (e) => {
        const copy = e.target.closest('#channels-table [data-copy]');
        if (copy) {
            copyToClipboard(copy.dataset.copy, copy);
            return;
        }
        const btn = e.target.closest('#channels-table [data-action]');
        if (!btn) return;
        const id = parseInt(btn.dataset.id);
        if (btn.dataset.action === 'edit') {
            showModal(id);
        } else if (btn.dataset.action === 'delete') {
            if (!confirm('确定要删除这个Channel吗？')) return;
            try {
                await adminRequest(`/channels/${id}`, { method: 'DELETE' });
                showAlert('Channel删除成功');
                load();
                refreshOverview();
            } catch (error) {
                showAlert('删除失败: ' + error.message, 'error');
            }
        }
    });
}

export async function load() {
    try {
        const data = await adminRequest('/channels');
        channels = data.data || [];
        const tbody = document.querySelector('#channels-table tbody');
        tbody.innerHTML = '';

        channels.forEach(ch => {
            const origins = (ch.allowed_origins || []).join(', ') || '-';
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td>${ch.id}</td>
                <td class="cell-channel" data-copy="${ch.channel_id}" title="点击复制完整Channel ID">${ch.channel_id}</td>
                <td>${ch.rate_limit}</td>
                <td class="cell-origins" title="${origins}">${origins}</td>
                <td><span class="${ch.status === 'active' ? 'status-active' : 'status-inactive'}">${ch.status}</span></td>
                <td>${formatDate(ch.created_at)}</td>
                <td>${formatDate(ch.last_used_at)}</td>
                <td>
                    <button class="btn btn-sm btn-primary" data-action="edit" data-id="${ch.id}">编辑</button>
                    <button class="btn btn-sm btn-danger" data-action="delete" data-id="${ch.id}">删除</button>
                </td>`;
            tbody.appendChild(tr);
        });
    } catch (error) {
        showAlert('加载Channels失败: ' + error.message, 'error');
    }
}

function copyToClipboard(text, element) {
    navigator.clipboard.writeText(text).then(() => {
        const original = element.textContent;
        element.textContent = '已复制!';
        element.style.color = 'green';
        setTimeout(() => {
            element.textContent = original;
            element.style.color = '';
        }, 1000);
    }).catch(err => {
        showAlert('复制失败: ' + err.message, 'error');
    });
}

function showModal(id = null) {
    document.getElementById('channel-result').style.display = 'none';
    document.getElementById('channel-submit').style.display = 'block';
    document.getElementById('channel-submit').textContent = '保存';

    if (id) {
        const ch = channels.find(c => c.id === id);
        if (ch) {
            document.getElementById('channel-modal-title').textContent = '编辑Channel';
            document.getElementById('channel-id').value = ch.id;
            document.getElementById('channel-key').value = ch.channel_id;
            document.getElementById('channel-ratelimit').value = ch.rate_limit;
            document.getElementById('channel-origins').value = (ch.allowed_origins || []).join('\n');
            document.getElementById('channel-status').value = ch.status;
            document.getElementById('channel-status-group').style.display = 'block';
        }
    } else {
        document.getElementById('channel-modal-title').textContent = '创建Channel';
        document.getElementById('channel-form').reset();
        document.getElementById('channel-id').value = '';
        document.getElementById('channel-ratelimit').value = '60';
        document.getElementById('channel-status-group').style.display = 'none';
    }

    document.getElementById('channel-modal').classList.add('active');
}

function parseOriginsInput(raw) {
    return raw.split(/[\n,]/).map(s => s.trim()).filter(s => s !== '');
}

async function onSubmit(e) {
    e.preventDefault();

    const id = document.getElementById('channel-id').value;
    const customID = document.getElementById('channel-key').value.trim();
    const data = {
        rate_limit: parseInt(document.getElementById('channel-ratelimit').value),
    };

    // 创建时可自定义 channel_id; 编辑时不可修改.
    if (customID && !id) {
        data.channel_id = customID;
    }
    const origins = parseOriginsInput(document.getElementById('channel-origins').value);
    if (origins.length > 0 || id) {
        data.allowed_origins = origins;
    }
    if (id) {
        data.status = document.getElementById('channel-status').value;
    }

    try {
        if (id) {
            await adminRequest(`/channels/${id}`, { method: 'PUT', body: JSON.stringify(data) });
            showAlert('Channel更新成功');
            closeModal('channel-modal');
        } else {
            const result = await adminRequest('/channels', { method: 'POST', body: JSON.stringify(data) });
            document.getElementById('channel-token').textContent = result.channel_id;
            document.getElementById('channel-result').style.display = 'block';
            document.getElementById('channel-submit').style.display = 'none';
            showAlert('Channel创建成功');
        }

        load();
        refreshOverview();
    } catch (error) {
        showAlert('操作失败: ' + error.message, 'error');
    }
}
