// import.js — 导入域: URL 列表批量导入与 GitHub 仓库导入 (F-012).
// GitHub 导入走两步: 预览 (匹配数 + 样例 URL + 抽检) -> 确认导入.
import { adminRequest, showAlert } from '../core.js';
import { getCategories } from './categories.js';
import { refreshOverview } from './app.js';

export function init() {
    document.getElementById('batch-import-form').addEventListener('submit', onBatchImport);
    document.getElementById('github-import-form').addEventListener('submit', onGithubImport);
    document.getElementById('gh-btn-preview').addEventListener('click', onPreview);

    // 图源模式切换: 自定义镜像时显示 Base URL 输入.
    document.querySelectorAll('input[name="gh-source-mode"]').forEach(radio => {
        radio.addEventListener('change', () => {
            document.getElementById('gh-mirror-group').style.display =
                radio.value === 'mirror' && radio.checked ? 'block' : 'none';
            resetPreview();
        });
    });

    // 链接/覆盖字段变化后预览失效.
    ['gh-repo-url', 'gh-path', 'gh-ref', 'gh-base-url'].forEach(id => {
        document.getElementById(id).addEventListener('input', resetPreview);
    });
}

export async function load() {
    const categories = await getCategories(true);
    const options = categories.map(cat => `<option value="${cat.id}">${cat.name}</option>`).join('');
    document.getElementById('import-category').innerHTML = options;
    document.getElementById('gh-category').innerHTML = options;
}

// 组装 GitHub 导入/预览公共参数.
function ghPayload() {
    const payload = {
        repo_url: document.getElementById('gh-repo-url').value.trim(),
        category_id: parseInt(document.getElementById('gh-category').value),
        auto_fetch: document.getElementById('gh-auto-fetch').checked,
    };
    const mode = document.querySelector('input[name="gh-source-mode"]:checked').value;
    if (mode === 'mirror') {
        payload.base_url = document.getElementById('gh-base-url').value.trim();
    }
    const p = document.getElementById('gh-path').value.trim();
    if (p) payload.path = p;
    const r = document.getElementById('gh-ref').value.trim();
    if (r) payload.ref = r;
    return payload;
}

function resetPreview() {
    document.getElementById('gh-preview').classList.remove('visible');
    document.getElementById('gh-btn-import').disabled = true;
}

async function onPreview() {
    const repoURL = document.getElementById('gh-repo-url').value.trim();
    if (!repoURL) {
        showAlert('请先填写仓库链接', 'error');
        return;
    }
    const mode = document.querySelector('input[name="gh-source-mode"]:checked').value;
    if (mode === 'mirror' && !document.getElementById('gh-base-url').value.trim()) {
        showAlert('自定义镜像需要填写 Base URL', 'error');
        return;
    }

    const panel = document.getElementById('gh-preview');
    const btn = document.getElementById('gh-btn-preview');
    btn.disabled = true;
    btn.textContent = '预览中...';
    resetPreview();

    try {
        const result = await adminRequest('/import/github/preview', {
            method: 'POST',
            body: JSON.stringify(ghPayload()),
        });

        document.getElementById('gh-preview-count').textContent = result.matched;

        const samples = document.getElementById('gh-preview-samples');
        samples.innerHTML = '';
        result.samples.forEach(u => {
            const li = document.createElement('li');
            li.textContent = u;
            li.title = '点击复制';
            li.addEventListener('click', () => navigator.clipboard.writeText(u));
            samples.appendChild(li);
        });

        const checks = document.getElementById('gh-preview-checks');
        checks.innerHTML = result.checks.map(ch =>
            `<div class="${ch.ok ? 'check-ok' : 'check-fail'}">${ch.ok ? '✓' : '✗'} ${ch.detail} — ${ch.url}</div>`
        ).join('');

        panel.classList.add('visible');

        if (result.matched === 0) {
            showAlert('未匹配到图片, 请检查子目录或分支', 'error');
            return;
        }
        // 校验门: 抽检全部失败时拒绝导入.
        if (result.probe_total > 0 && result.probe_ok === 0) {
            document.getElementById('gh-btn-import').disabled = true;
            showAlert('抽检全部失败, 图源地址大概率拼错, 已禁止导入', 'error');
            return;
        }
        if (result.probe_ok < result.probe_total) {
            showAlert(`抽检 ${result.probe_ok}/${result.probe_total} 通过, 请检查失败样例后再导入`, 'error');
        }
        document.getElementById('gh-btn-import').disabled = false;
    } catch (error) {
        showAlert('预览失败: ' + error.message, 'error');
    } finally {
        btn.disabled = false;
        btn.textContent = '预览';
    }
}

async function onGithubImport(e) {
    e.preventDefault();
    const resultBox = document.getElementById('github-import-result');
    resultBox.textContent = '导入中, 仓库较大时可能需要数十秒...';

    try {
        const result = await adminRequest('/import/github', {
            method: 'POST',
            body: JSON.stringify(ghPayload()),
        });
        showAlert(`GitHub 导入完成: 新增 ${result.imported}, 跳过 ${result.skipped}`);
        resultBox.textContent =
            `新增 ${result.imported} 张, 跳过 ${result.skipped} 张 (已存在), 元数据补全排队 ${result.fetch_pending}`;
        resetPreview();
        refreshOverview();
    } catch (error) {
        resultBox.textContent = '';
        showAlert('GitHub 导入失败: ' + error.message, 'error');
    }
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
