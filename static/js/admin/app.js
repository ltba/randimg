// app.js — admin 入口: 概览统计, 导航切换, 各域模块调度.
import { adminRequest } from '../core.js';
import * as images from './images.js';
import * as categories from './categories.js';
import * as channels from './channels.js';
import * as stats from './stats.js';
import * as importer from './import.js';

const sectionLoaders = {
    images: images.load,
    categories: categories.load,
    channels: channels.load,
    stats: stats.load,
    import: importer.load,
};

export async function refreshOverview() {
    try {
        const data = await adminRequest('/stats/overview');
        document.getElementById('stat-images').textContent = data.total_images;
        document.getElementById('stat-keys').textContent = data.active_channels;
        document.getElementById('stat-today').textContent = data.today_calls;
        document.getElementById('stat-total').textContent = data.total_calls;
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

function bindNav() {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const section = btn.dataset.section;
            document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
            document.getElementById(`section-${section}`).classList.add('active');
            sectionLoaders[section]?.();
        });
    });
}

images.init();
categories.init();
channels.init();
stats.init();
importer.init();
bindNav();
refreshOverview();
images.load();
