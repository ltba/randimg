// gallery.js — 画廊页: 无限滚动, 懒加载, lightbox; 公开接口经 core 请求层.
import { pubRequest } from './core.js';

let currentPage = 1;
let isLoading = false;
let hasMore = true;
let allImages = [];
let currentLightboxIndex = 0;

const imageObserver = new IntersectionObserver((entries, observer) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            const img = entry.target;
            const src = img.getAttribute('data-src');
            if (src) {
                img.src = src;
                img.removeAttribute('data-src');
                observer.unobserve(img);
            }
        }
    });
}, {
    rootMargin: '50px'
});

const scrollObserver = new IntersectionObserver((entries) => {
    if (entries.some(entry => entry.isIntersecting) && !isLoading && hasMore) {
        loadImages(currentPage + 1, true);
    }
}, {
    rootMargin: '200px'
});

async function loadCategories() {
    try {
        const categories = await pubRequest('/categories');
        const select = document.getElementById('category-filter');
        categories.forEach(cat => {
            const option = document.createElement('option');
            option.value = cat.slug;
            option.textContent = cat.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

async function loadImages(page = 1, append = false) {
    if (isLoading) return;
    isLoading = true;

    const loadingIndicator = document.getElementById('loading-indicator');
    loadingIndicator.classList.remove('hidden');

    try {
        const category = document.getElementById('category-filter').value;
        const device = document.getElementById('device-filter').value;

        let url = `/images?page=${page}&page_size=20`;
        if (category) url += `&category=${encodeURIComponent(category)}`;
        if (device) url += `&device=${device}`;

        const data = await pubRequest(url);

        if (!append) {
            allImages = [];
            currentPage = 1;
        }

        allImages = allImages.concat(data.data);
        renderImages(data.data, append);

        hasMore = data.pagination.page < data.pagination.total_page;
        currentPage = page;
    } catch (error) {
        console.error('Failed to load images:', error);
    } finally {
        isLoading = false;
        loadingIndicator.classList.add('hidden');
    }
}

function renderImages(images, append = false) {
    const grid = document.getElementById('gallery-grid');
    if (!append) {
        grid.innerHTML = '';
    }

    images.forEach((image) => {
        const item = document.createElement('div');
        item.className = 'gallery-item';
        item.addEventListener('click', () => openLightbox(allImages.indexOf(image)));

        const img = document.createElement('img');
        img.setAttribute('data-src', image.source_url);
        img.alt = image.category?.name || 'Image';
        img.loading = 'lazy';
        imageObserver.observe(img);

        const info = document.createElement('div');
        info.className = 'gallery-item-info';

        const category = document.createElement('span');
        category.className = 'category';
        category.textContent = image.category?.name || '未分类';

        const dimensions = document.createElement('div');
        dimensions.className = 'dimensions';
        dimensions.textContent = image.width && image.height
            ? `${image.width} × ${image.height}`
            : '尺寸未知';

        info.appendChild(category);
        info.appendChild(dimensions);
        item.appendChild(img);
        item.appendChild(info);
        grid.appendChild(item);
    });

    const items = grid.querySelectorAll('.gallery-item');
    if (items.length > 0) {
        scrollObserver.observe(items[items.length - 1]);
    }
}

function openLightbox(index) {
    currentLightboxIndex = index;
    document.getElementById('lightbox-img').src = allImages[index].source_url;
    document.getElementById('lightbox').classList.add('active');
    document.addEventListener('keydown', handleLightboxKeyboard);
}

function closeLightbox() {
    document.getElementById('lightbox').classList.remove('active');
    document.removeEventListener('keydown', handleLightboxKeyboard);
}

function navigateLightbox(direction) {
    currentLightboxIndex += direction;
    if (currentLightboxIndex < 0) {
        currentLightboxIndex = allImages.length - 1;
    } else if (currentLightboxIndex >= allImages.length) {
        currentLightboxIndex = 0;
    }
    document.getElementById('lightbox-img').src = allImages[currentLightboxIndex].source_url;
}

function handleLightboxKeyboard(e) {
    if (e.key === 'Escape') {
        closeLightbox();
    } else if (e.key === 'ArrowLeft') {
        navigateLightbox(-1);
    } else if (e.key === 'ArrowRight') {
        navigateLightbox(1);
    }
}

document.getElementById('category-filter').addEventListener('change', () => loadImages(1, false));
document.getElementById('device-filter').addEventListener('change', () => loadImages(1, false));
document.getElementById('lightbox-close').addEventListener('click', closeLightbox);
document.getElementById('lightbox-prev').addEventListener('click', () => navigateLightbox(-1));
document.getElementById('lightbox-next').addEventListener('click', () => navigateLightbox(1));
document.getElementById('lightbox').addEventListener('click', (e) => {
    if (e.target.id === 'lightbox') {
        closeLightbox();
    }
});

loadCategories();
loadImages();
