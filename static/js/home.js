// home.js — 首页: 公开统计, 随机背景, 背景刷新按钮.
import { pubRequest } from './core.js';

async function loadStats() {
    try {
        const data = await pubRequest('/stats');
        document.getElementById('stat-images').textContent = data.total_images || 0;
        document.getElementById('stat-calls').textContent = data.total_calls || 0;
        document.getElementById('stat-today').textContent = data.today_calls || 0;
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

function setRandomBackground() {
    document.getElementById('background').style.backgroundImage = `url('/api/random?category=acg')`;
}

function refreshBackground() {
    const bg = document.getElementById('background');
    bg.style.opacity = '0';
    setTimeout(() => {
        setRandomBackground();
        bg.style.transition = 'opacity 0.5s';
        bg.style.opacity = '1';
    }, 500);
}

document.querySelector('.refresh-bg').addEventListener('click', refreshBackground);

loadStats();
setRandomBackground();
