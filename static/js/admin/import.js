// import.js — 导入域: URL 列表批量导入与 GitHub 仓库导入 (F-012).
import { adminRequest, showAlert } from '../core.js';
import { getCategories } from './categories.js';
import { refreshOverview } from './app.js';

export function init() {
    document.getElementById('batch-import-form').addEventListener('submit', onBatchImport);
    document.getElementById('github-import-form').addEventListener('submit', onGithubImport);
}

export async function load() {
    const categories = await getCategories(true);
    const options = categories.map(cat => `<option value="${cat.id}">${cat.name}</option>`).join('');
    document.getElementById('import-category').innerHTML = options;
    document.getElementById('gh-category').innerHTML = options;
}

async function onBatchImport(e) {
    e.preventDefault();
    const urls = document.getElementById('import-urls').value
        .split(/[\n,]/).map(s => s.trim()).filter(s => s !== '');
    if (urls.length === 0) {
        showAlert('请粘贴至少一个图片URL', 'error');
        return;
    }
    if (urls.length > 1000) {
        showAlert('单次最多 1000 条', 'error');
        return;
    }

    const categoryId = parseInt(document.getElementById('import-category').value);
    const autoFetch = document.getElementById('import-auto-fetch').checked;

    try {
        const result = await adminRequest('/images/batch', {
            method: 'POST',
            body: JSON.stringify({
                images: urls.map(u => ({
                    source_url: u,
                    category_id: categoryId,
                    auto_fetch: autoFetch,
                })),
            }),
        });
        showAlert(`批量导入成功！共添加 ${result.count} 张图片`);
        document.getElementById('batch-import-result').textContent =
            `已导入 ${result.count} 张 (元数据补全排队 ${result.fetch_pending ?? 0})`;
        document.getElementById('import-urls').value = '';
        refreshOverview();
    } catch (error) {
        showAlert('批量导入失败: ' + error.message, 'error');
    }
}

async function onGithubImport(e) {
    e.preventDefault();
    const payload = {
        owner: document.getElementById('gh-owner').value.trim(),
        repo: document.getElementById('gh-repo').value.trim(),
        category_id: parseInt(document.getElementById('gh-category').value),
    };
    const path = document.getElementById('gh-path').value.trim();
    if (path) payload.path = path;
    const ref = document.getElementById('gh-ref').value.trim();
    if (ref) payload.ref = ref;
    const baseURL = document.getElementById('gh-base-url').value.trim();
    if (baseURL) payload.base_url = baseURL;
    payload.auto_fetch = document.getElementById('gh-auto-fetch').checked;

    const resultBox = document.getElementById('github-import-result');
    resultBox.textContent = '导入中, 仓库较大时可能需要数十秒...';

    try {
        const result = await adminRequest('/import/github', {
            method: 'POST',
            body: JSON.stringify(payload),
        });
        showAlert(`GitHub 导入完成: 新增 ${result.imported}, 跳过 ${result.skipped}`);
        resultBox.textContent =
            `新增 ${result.imported} 张, 跳过 ${result.skipped} 张 (已存在), 元数据补全排队 ${result.fetch_pending}`;
        refreshOverview();
    } catch (error) {
        resultBox.textContent = '';
        showAlert('GitHub 导入失败: ' + error.message, 'error');
    }
}
