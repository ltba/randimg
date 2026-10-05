// 使用 js/api.js 提供的 API 工具
const { request: apiRequest, getToken: getAdminToken, clearToken: clearAdminToken } = window.API;
const { showAlert: showAlertUtil, formatDate: formatDateUtil } = window.Utils;

// 全局状态
let currentPage = 1;
let categories = [];

// 工具函数
function showAlert(message, type = 'success') {
    const container = document.getElementById('alert-container');
    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;
    container.appendChild(alert);

    setTimeout(() => {
        alert.remove();
    }, 3000);
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN');
}

// 复制到剪贴板
function copyToClipboard(text, element) {
    navigator.clipboard.writeText(text).then(() => {
        const originalText = element.textContent;
        element.textContent = '已复制!';
        element.style.color = 'green';
        setTimeout(() => {
            element.textContent = originalText;
            element.style.color = '';
        }, 1000);
    }).catch(err => {
        showAlert('复制失败: ' + err.message, 'error');
    });
}

// 导航切换
document.querySelectorAll('.nav-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        const section = btn.dataset.section;

        // 更新导航状态
        document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        // 更新内容区域
        document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
        document.getElementById(`section-${section}`).classList.add('active');

        // 加载对应数据
        switch(section) {
            case 'images':
                loadImages();
                break;
            case 'categories':
                loadCategories();
                break;
            case 'channels':
                loadChannels();
                break;
            case 'stats':
                loadStatsChannels();
                break;
        }
    });
});

// 加载统计概览
async function loadStatsOverview() {
    try {
        const data = await apiRequest('/stats/overview');
        document.getElementById('stat-images').textContent = data.total_images;
        document.getElementById('stat-keys').textContent = data.active_channels;
        document.getElementById('stat-today').textContent = data.today_calls;
        document.getElementById('stat-total').textContent = data.total_calls;
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// ========== 图片管理 ==========
let selectedImages = new Set();

async function loadImages(page = 1) {
    try {
        const data = await apiRequest(`/images?page=${page}&page_size=20`);
        const tbody = document.querySelector('#images-table tbody');
        tbody.innerHTML = '';

        data.data.forEach(image => {
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td><input type="checkbox" class="image-checkbox" value="${image.id}" onchange="updateSelection()"></td>
                <td>${image.id}</td>
                <td><img src="${image.source_url}" style="max-width: 60px; max-height: 60px; object-fit: cover; border-radius: 4px;"></td>
                <td style="max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${image.source_url}">${image.source_url}</td>
                <td>${image.width || '-'} x ${image.height || '-'}</td>
                <td>${image.category ? image.category.name : '-'}</td>
                <td>${image.source || '-'}</td>
                <td><span style="color: ${image.status === 'active' ? 'green' : 'red'}">${image.status}</span></td>
                <td>
                    <button class="btn btn-sm btn-primary" onclick="editImage(${image.id})">编辑</button>
                    <button class="btn btn-sm btn-danger" onclick="deleteImage(${image.id})">删除</button>
                </td>
            `;
            tbody.appendChild(tr);
        });

        // 分页
        renderPagination('images-pagination', data.pagination, loadImages);
        currentPage = page;
        updateSelection();
    } catch (error) {
        showAlert('加载图片失败: ' + error.message, 'error');
    }
}

function renderPagination(containerId, pagination, loadFunc) {
    const container = document.getElementById(containerId);
    container.innerHTML = '';

    const totalPages = pagination.total_page;
    const currentPage = pagination.page;

    if (totalPages <= 1) return;

    // 上一页按钮
    if (currentPage > 1) {
        const prevBtn = document.createElement('button');
        prevBtn.textContent = '«';
        prevBtn.onclick = () => loadFunc(currentPage - 1);
        container.appendChild(prevBtn);
    }

    // 智能显示页码：当前页前后各2页
    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);

    // 第一页
    if (startPage > 1) {
        const btn = document.createElement('button');
        btn.textContent = '1';
        btn.onclick = () => loadFunc(1);
        container.appendChild(btn);

        if (startPage > 2) {
            const dots = document.createElement('span');
            dots.textContent = '...';
            dots.style.padding = '0 10px';
            container.appendChild(dots);
        }
    }

    // 中间页码
    for (let i = startPage; i <= endPage; i++) {
        const btn = document.createElement('button');
        btn.textContent = i;
        btn.className = i === currentPage ? 'active' : '';
        btn.onclick = () => loadFunc(i);
        container.appendChild(btn);
    }

    // 最后一页
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            const dots = document.createElement('span');
            dots.textContent = '...';
            dots.style.padding = '0 10px';
            container.appendChild(dots);
        }

        const btn = document.createElement('button');
        btn.textContent = totalPages;
        btn.onclick = () => loadFunc(totalPages);
        container.appendChild(btn);
    }

    // 下一页按钮
    if (currentPage < totalPages) {
        const nextBtn = document.createElement('button');
        nextBtn.textContent = '»';
        nextBtn.onclick = () => loadFunc(currentPage + 1);
        container.appendChild(nextBtn);
    }
}

async function showImageModal(id = null) {
    // 加载分类列表
    if (categories.length === 0) {
        categories = await apiRequest('/categories');
    }

    const select = document.getElementById('image-category');
    select.innerHTML = categories.map(cat =>
        `<option value="${cat.id}">${cat.name}</option>`
    ).join('');

    if (id) {
        // 编辑模式
        const image = await apiRequest(`/images/${id}`);
        document.getElementById('image-modal-title').textContent = '编辑图片';
        document.getElementById('image-id').value = image.id;
        document.getElementById('image-url').value = image.source_url;
        document.getElementById('image-category').value = image.category_id;
        document.getElementById('image-width').value = image.width || '';
        document.getElementById('image-height').value = image.height || '';
        document.getElementById('image-format').value = image.format || '';
        document.getElementById('image-source').value = image.source || '';
    } else {
        // 新建模式
        document.getElementById('image-modal-title').textContent = '添加图片';
        document.getElementById('image-form').reset();
        document.getElementById('image-id').value = '';
    }

    document.getElementById('image-modal').classList.add('active');
}

document.getElementById('image-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const id = document.getElementById('image-id').value;
    const autoFetch = document.getElementById('image-auto-fetch').checked;
    const data = {
        source_url: document.getElementById('image-url').value,
        category_id: parseInt(document.getElementById('image-category').value),
        width: document.getElementById('image-width').value ? parseInt(document.getElementById('image-width').value) : null,
        height: document.getElementById('image-height').value ? parseInt(document.getElementById('image-height').value) : null,
        format: document.getElementById('image-format').value || null,
        source: document.getElementById('image-source').value || null,
        auto_fetch: autoFetch,
    };

    try {
        if (id) {
            await apiRequest(`/images/${id}`, {
                method: 'PUT',
                body: JSON.stringify(data)
            });
            showAlert('图片更新成功');
        } else {
            await apiRequest('/images', {
                method: 'POST',
                body: JSON.stringify(data)
            });

            showAlert('图片添加成功');
        }

        closeModal('image-modal');
        loadImages(currentPage);
        loadStatsOverview();
    } catch (error) {
        showAlert('操作失败: ' + error.message, 'error');
    }
});

async function editImage(id) {
    await showImageModal(id);
}

async function deleteImage(id) {
    if (!confirm('确定要删除这张图片吗？')) return;

    try {
        await apiRequest(`/images/${id}`, { method: 'DELETE' });
        showAlert('图片删除成功');
        loadImages(currentPage);
        loadStatsOverview();
    } catch (error) {
        showAlert('删除失败: ' + error.message, 'error');
    }
}

// ========== 分类管理 ==========
async function loadCategories() {
    try {
        categories = await apiRequest('/categories');
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
                    <button class="btn btn-sm btn-primary" onclick="editCategory(${cat.id})">编辑</button>
                    <button class="btn btn-sm btn-danger" onclick="deleteCategory(${cat.id})">删除</button>
                </td>
            `;
            tbody.appendChild(tr);
        });
    } catch (error) {
        showAlert('加载分类失败: ' + error.message, 'error');
    }
}

function showCategoryModal(id = null) {
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

document.getElementById('category-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const id = document.getElementById('category-id').value;
    const data = {
        name: document.getElementById('category-name').value,
        slug: document.getElementById('category-slug').value,
        description: document.getElementById('category-description').value || null,
    };

    try {
        if (id) {
            await apiRequest(`/categories/${id}`, {
                method: 'PUT',
                body: JSON.stringify(data)
            });
            showAlert('分类更新成功');
        } else {
            await apiRequest('/categories', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            showAlert('分类添加成功');
        }

        closeModal('category-modal');
        loadCategories();
    } catch (error) {
        showAlert('操作失败: ' + error.message, 'error');
    }
});

async function editCategory(id) {
    showCategoryModal(id);
}

async function deleteCategory(id) {
    if (!confirm('确定要删除这个分类吗？')) return;

    try {
        await apiRequest(`/categories/${id}`, { method: 'DELETE' });
        showAlert('分类删除成功');
        loadCategories();
    } catch (error) {
        showAlert('删除失败: ' + error.message, 'error');
    }
}

// ========== Channel管理 ==========
let channelsList = [];

async function loadChannels() {
    try {
        const data = await apiRequest('/channels');
        channelsList = data.data || [];
        const tbody = document.querySelector('#channels-table tbody');
        tbody.innerHTML = '';

        channelsList.forEach(ch => {
            const origins = (ch.allowed_origins || []).join(', ') || '-';
            const tr = document.createElement('tr');
            tr.innerHTML = `
                <td>${ch.id}</td>
                <td style="font-family: monospace; font-size: 12px; max-width: 260px; overflow: hidden; text-overflow: ellipsis; cursor: pointer;"
                    title="点击复制完整Channel ID"
                    onclick="copyToClipboard('${ch.channel_id}', this)">
                    ${ch.channel_id}
                </td>
                <td>${ch.rate_limit}</td>
                <td style="max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${origins}">${origins}</td>
                <td data-status="${ch.status}"><span style="color: ${ch.status === 'active' ? 'green' : 'red'}">${ch.status}</span></td>
                <td>${formatDate(ch.created_at)}</td>
                <td>${formatDate(ch.last_used_at)}</td>
                <td>
                    <button class="btn btn-sm btn-primary" onclick="editChannel(${ch.id})">编辑</button>
                    <button class="btn btn-sm btn-danger" onclick="deleteChannel(${ch.id})">删除</button>
                </td>
            `;
            tbody.appendChild(tr);
        });
    } catch (error) {
        showAlert('加载Channels失败: ' + error.message, 'error');
    }
}

function parseOriginsInput(raw) {
    return raw.split(/[\n,]/).map(s => s.trim()).filter(s => s !== '');
}

function showChannelModal(id = null) {
    document.getElementById('channel-result').style.display = 'none';
    document.getElementById('channel-submit').style.display = 'block';
    document.getElementById('channel-submit').textContent = '保存';

    if (id) {
        const ch = channelsList.find(c => c.id === id);
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
        document.getElementById('channel-key').value = '';
        document.getElementById('channel-ratelimit').value = '60';
        document.getElementById('channel-origins').value = '';
        document.getElementById('channel-status-group').style.display = 'none';
    }

    document.getElementById('channel-modal').classList.add('active');
}

document.getElementById('channel-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const id = document.getElementById('channel-id').value;
    const customID = document.getElementById('channel-key').value.trim();
    const data = {
        rate_limit: parseInt(document.getElementById('channel-ratelimit').value)
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
            await apiRequest(`/channels/${id}`, {
                method: 'PUT',
                body: JSON.stringify(data)
            });
            showAlert('Channel更新成功');
            closeModal('channel-modal');
        } else {
            const result = await apiRequest('/channels', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            document.getElementById('channel-token').textContent = result.channel_id;
            document.getElementById('channel-result').style.display = 'block';
            document.getElementById('channel-submit').style.display = 'none';
            showAlert('Channel创建成功');
        }

        loadChannels();
        loadStatsOverview();
    } catch (error) {
        showAlert('操作失败: ' + error.message, 'error');
    }
});

async function editChannel(id) {
    showChannelModal(id);
}

async function deleteChannel(id) {
    if (!confirm('确定要删除这个Channel吗？')) return;

    try {
        await apiRequest(`/channels/${id}`, { method: 'DELETE' });
        showAlert('Channel删除成功');
        loadChannels();
        loadStatsOverview();
    } catch (error) {
        showAlert('删除失败: ' + error.message, 'error');
    }
}

// ========== 统计数据 ==========
async function loadStatsChannels() {
    try {
        const data = await apiRequest('/channels');
        const channels = data.data || [];
        const select = document.getElementById('stats-channel');
        select.innerHTML = '<option value="">请选择</option>' +
            channels.map(ch => `<option value="${ch.id}">${ch.channel_id.substring(0, 24)}... (ID: ${ch.id})</option>`).join('');
    } catch (error) {
        showAlert('加载Channels失败: ' + error.message, 'error');
    }
}

async function loadStats() {
    const channelId = document.getElementById('stats-channel').value;
    if (!channelId) {
        document.getElementById('stats-table').style.display = 'none';
        document.getElementById('stats-summary').style.display = 'none';
        return;
    }

    try {
        const data = await apiRequest(`/stats?channel_id=${channelId}`);
        const tbody = document.querySelector('#stats-table tbody');
        tbody.innerHTML = '';

        if (data.logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="2" style="text-align: center; color: #666;">暂无数据 (统计异步落库, 最多延迟约10秒)</td></tr>';
        } else {
            data.logs.forEach(log => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${formatDate(log.created_at)}</td>
                    <td>${log.channel_id}</td>
                `;
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

// ========== 脚本工具 ==========

// 脚本日志输出
function scriptLog(message, type = 'info') {
    const logDiv = document.getElementById('script-log');
    const timestamp = new Date().toLocaleTimeString();
    let color = '#fff';

    switch(type) {
        case 'success':
            color = '#68d391';
            break;
        case 'error':
            color = '#fc8181';
            break;
        case 'warning':
            color = '#f6ad55';
            break;
    }

    const logEntry = document.createElement('div');
    logEntry.style.color = color;
    logEntry.textContent = `[${timestamp}] ${message}`;
    logDiv.appendChild(logEntry);
    logDiv.scrollTop = logDiv.scrollHeight;
}

// 清空日志
function clearScriptLog() {
    const logDiv = document.getElementById('script-log');
    logDiv.innerHTML = '<div style="color: #68d391;">控制台输出：</div>';
}

// 脚本环境中可用的函数
const scriptEnv = {
    // 添加图片
    addImage: async function(data) {
        return await apiRequest('/images', {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    // 获取分类列表
    getCategories: async function() {
        return await apiRequest('/categories');
    },

    // 延迟函数
    sleep: function(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    },

    // 日志函数
    console: {
        log: (...args) => scriptLog(args.join(' '), 'info'),
        error: (...args) => scriptLog(args.join(' '), 'error'),
        warn: (...args) => scriptLog(args.join(' '), 'warning'),
        info: (...args) => scriptLog(args.join(' '), 'info'),
    }
};

// 执行脚本
async function executeScript() {
    const scriptCode = document.getElementById('script-editor').value;
    clearScriptLog();
    scriptLog('开始执行脚本...', 'info');

    try {
        // 创建异步函数并执行
        const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
        const scriptFunc = new AsyncFunction(
            'addImage',
            'getCategories',
            'sleep',
            'console',
            'getAdminToken',
            'fetch',
            scriptCode
        );

        await scriptFunc(
            scriptEnv.addImage,
            scriptEnv.getCategories,
            scriptEnv.sleep,
            scriptEnv.console,
            getAdminToken,
            fetch.bind(window)
        );

        scriptLog('脚本执行完成', 'success');
        loadImages(currentPage);
        loadStatsOverview();
    } catch (error) {
        scriptLog('脚本执行错误: ' + error.message, 'error');
        console.error(error);
    }
}

// 加载脚本示例
function loadScriptExample(type) {
    const editor = document.getElementById('script-editor');

    if (type === 'simple') {
        editor.value = `// 简单批量添加示例
const categories = await getCategories();
console.log('可用分类:', categories.map(c => c.name + ' (' + c.slug + ')').join(', '));

// 选择第一个分类
const category = categories[0];
if (!category) {
    console.error('没有可用分类');
    return;
}

// 图片列表
const images = [
    { url: "https://picsum.photos/800/600?random=1", source: "Lorem Picsum" },
    { url: "https://picsum.photos/800/600?random=2", source: "Lorem Picsum" },
    { url: "https://picsum.photos/800/600?random=3", source: "Lorem Picsum" },
];

// 使用批量API一次性添加
try {
    const response = await fetch('/api/admin/images/batch', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + getAdminToken()
        },
        body: JSON.stringify({
            images: images.map(img => ({
                source_url: img.url,
                category_id: category.id,
                source: img.source,
                auto_fetch: true
            }))
        })
    });

    if (!response.ok) {
        throw new Error('HTTP ' + response.status);
    }

    const result = await response.json();
    console.log('✓ 批量添加成功！共添加 ' + result.count + ' 张图片');
} catch (error) {
    console.error('✗ 批量添加失败:', error.message);
}`;
    } else if (type === 'unsplash') {
        editor.value = `// Unsplash API 批量导入示例
// 需要先在 https://unsplash.com/developers 申请 Access Key

const UNSPLASH_ACCESS_KEY = 'YOUR_ACCESS_KEY_HERE';
const QUERY = 'landscape'; // 搜索关键词
const COUNT = 10; // 获取数量

// 获取分类
const categories = await getCategories();
const targetCategory = categories.find(c => c.slug === 'landscape') || categories[0];

if (!targetCategory) {
    console.error('没有可用分类');
    return;
}

console.log('使用分类:', targetCategory.name);

// 从 Unsplash 获取随机图片
const response = await fetch(
    'https://api.unsplash.com/photos/random?count=' + COUNT + '&query=' + QUERY,
    {
        headers: {
            'Authorization': 'Client-ID ' + UNSPLASH_ACCESS_KEY
        }
    }
);

if (!response.ok) {
    console.error('Unsplash API 请求失败');
    return;
}

const photos = await response.json();
console.log('获取到 ' + photos.length + ' 张图片');

// 使用批量API一次性添加
try {
    const batchResponse = await fetch('/api/admin/images/batch', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + getAdminToken()
        },
        body: JSON.stringify({
            images: photos.map(photo => ({
                source_url: photo.urls.regular,
                category_id: targetCategory.id,
                width: photo.width,
                height: photo.height,
                format: 'jpeg',
                source: 'Unsplash - ' + photo.user.name,
                auto_fetch: false
            }))
        })
    });

    if (!batchResponse.ok) {
        throw new Error('HTTP ' + batchResponse.status);
    }

    const result = await batchResponse.json();
    console.log('✓ 批量添加成功！共添加 ' + result.count + ' 张图片');
} catch (error) {
    console.error('✗ 批量添加失败:', error.message);
}`;
    }
}

// ========== 批量操作 ==========

// 更新选择状态
function updateSelection() {
    const checkboxes = document.querySelectorAll('.image-checkbox:checked');
    selectedImages.clear();
    checkboxes.forEach(cb => selectedImages.add(parseInt(cb.value)));

    const count = selectedImages.size;
    document.getElementById('selected-count').textContent = count;
    document.getElementById('batch-actions').style.display = count > 0 ? 'block' : 'none';
}

// 全选/取消全选
function toggleSelectAll() {
    const selectAll = document.getElementById('select-all-images').checked;
    document.querySelectorAll('.image-checkbox').forEach(cb => {
        cb.checked = selectAll;
    });
    updateSelection();
}

// 清空选择
function clearSelection() {
    document.getElementById('select-all-images').checked = false;
    document.querySelectorAll('.image-checkbox').forEach(cb => {
        cb.checked = false;
    });
    updateSelection();
}

// 显示批量更新模态框
async function showBatchUpdateModal() {
    if (selectedImages.size === 0) {
        showAlert('请先选择要修改的图片', 'error');
        return;
    }

    // 加载分类列表
    if (categories.length === 0) {
        categories = await apiRequest('/categories');
    }

    const select = document.getElementById('batch-category');
    select.innerHTML = '<option value="">不修改</option>' +
        categories.map(cat => `<option value="${cat.id}">${cat.name}</option>`).join('');

    document.getElementById('batch-update-count').textContent = selectedImages.size;
    document.getElementById('batch-update-form').reset();
    document.getElementById('batch-update-modal').classList.add('active');
}

// 批量更新表单提交
document.getElementById('batch-update-form').addEventListener('submit', async (e) => {
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
        const result = await apiRequest('/images/batch', {
            method: 'PUT',
            body: JSON.stringify({
                image_ids: Array.from(selectedImages),
                updates: updates
            })
        });

        showAlert(`批量修改成功！已更新 ${result.updated} 张图片`);
        closeModal('batch-update-modal');
        clearSelection();
        loadImages(currentPage);
    } catch (error) {
        showAlert('批量修改失败: ' + error.message, 'error');
    }
});

// 批量删除
async function batchDeleteImages() {
    if (selectedImages.size === 0) {
        showAlert('请先选择要删除的图片', 'error');
        return;
    }

    if (!confirm(`确定要删除选中的 ${selectedImages.size} 张图片吗？`)) return;

    try {
        const result = await apiRequest('/images/batch', {
            method: 'DELETE',
            body: JSON.stringify({
                image_ids: Array.from(selectedImages)
            })
        });

        showAlert(`批量删除成功！已删除 ${result.deleted} 张图片`);
        clearSelection();
        loadImages(currentPage);
        loadStatsOverview();
    } catch (error) {
        showAlert('批量删除失败: ' + error.message, 'error');
    }
}

// 初始化
loadStatsOverview();
loadImages();
