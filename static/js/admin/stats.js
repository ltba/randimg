// stats.js — 统计数据域: 按 Channel 查询调用日志.
import { adminRequest, showAlert, formatDate } from '../core.js';

export function init() {
    document.getElementById('stats-channel').addEventListener('change', load);
}

export async function load() {
    // 频道下拉选项由本域维护 (显示 channel_id 摘要).
    const select = document.getElementById('stats-channel');
    try {
        const data = await adminRequest('/channels');
        const channels = data.data || [];
        const current = select.value;
        select.innerHTML = '<option value="">请选择</option>' +
            channels.map(ch => `<option value="${ch.id}">${ch.channel_id.substring(0, 24)}... (ID: ${ch.id})</option>`).join('');
        select.value = current;
    } catch (error) {
        showAlert('加载Channels失败: ' + error.message, 'error');
        return;
    }

    const channelId = select.value;
    if (!channelId) {
        document.getElementById('stats-table').style.display = 'none';
        document.getElementById('stats-summary').style.display = 'none';
        return;
    }

    try {
        const data = await adminRequest(`/stats?channel_id=${channelId}`);
        const tbody = document.querySelector('#stats-table tbody');
        tbody.innerHTML = '';

        if (data.logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="2" class="text-center" style="color: #666;">暂无数据 (统计异步落库, 最多延迟约10秒)</td></tr>';
        } else {
            data.logs.forEach(log => {
                const tr = document.createElement('tr');
                tr.innerHTML = `<td>${formatDate(log.created_at)}</td><td>${log.channel_id}</td>`;
                tbody.appendChild(tr);
            });
        }

        document.getElementById('stats-total').textContent = data.total || data.logs.length;
        document.getElementById('stats-table').style.display = 'table';
        document.getElementById('stats-summary').style.display = 'block';
    } catch (error) {
        showAlert('加载统计数据失败: ' + error.message, 'error');
    }
}
