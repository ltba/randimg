// Gallery page with lazy loading
const API_BASE = '/api';

let currentPage = 1;
let isLoading = false;
let hasMore = true;
let allImages = [];
let currentLightboxIndex = 0;

// Intersection Observer for lazy loading
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

// Infinite scroll observer
const scrollObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting && !isLoading && hasMore) {
            loadMoreImages();
        }
    });
}, {
    rootMargin: '200px'
});

// API request helper (同域名下不需要API key)
async function apiRequest(url, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers
    };

    const response = await fetch(API_BASE + url, {
        ...options,
        headers
    });

    if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
    }

    return await response.json();
}

// Load categories
async function loadCategories() {
    try {
        const categories = await apiRequest('/categories');
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

// Load images
async function loadImages(page = 1, append = false) {
    if (isLoading) return;
    isLoading = true;

    const loadingIndicator = document.getElementById('loading-indicator');
    loadingIndicator.classList.remove('hidden');

    try {
        const category = document.getElementById('category-filter').value;
        const device = document.getElementById('device-filter').value;

        let url = `/images?page=${page}&page_size=20`;
        if (category) url += `&category=${category}`;
        if (device) url += `&device=${device}`;

        const data = await apiRequest(url);

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

// Load more images (infinite scroll)
async function loadMoreImages() {
    await loadImages(currentPage + 1, true);
}

// Render images
function renderImages(images, append = false) {
    const grid = document.getElementById('gallery-grid');

    if (!append) {
        grid.innerHTML = '';
    }

    images.forEach((image, index) => {
        const item = document.createElement('div');
        item.className = 'gallery-item';
        item.onclick = () => openLightbox(allImages.indexOf(image));

        const img = document.createElement('img');
        img.setAttribute('data-src', image.source_url);
        img.alt = image.category?.name || 'Image';
        img.loading = 'lazy';

        // Observe for lazy loading
        imageObserver.observe(img);

        const info = document.createElement('div');
        info.className = 'gallery-item-info';

        const category = document.createElement('span');
        category.className = 'category';
        category.textContent = image.category?.name || '未分类';

        const dimensions = document.createElement('div');
        dimensions.className = 'dimensions';
        if (image.width && image.height) {
            dimensions.textContent = `${image.width} × ${image.height}`;
        } else {
            dimensions.textContent = '尺寸未知';
        }

        info.appendChild(category);
        info.appendChild(dimensions);

        item.appendChild(img);
        item.appendChild(info);
        grid.appendChild(item);
    });

    // Observe the last item for infinite scroll
    const items = grid.querySelectorAll('.gallery-item');
    if (items.length > 0) {
        scrollObserver.observe(items[items.length - 1]);
    }
}

// Lightbox functions
function openLightbox(index) {
    currentLightboxIndex = index;
    const lightbox = document.getElementById('lightbox');
    const img = document.getElementById('lightbox-img');
    img.src = allImages[index].source_url;
    lightbox.classList.add('active');

    // Keyboard navigation
    document.addEventListener('keydown', handleLightboxKeyboard);
}

function closeLightbox() {
    const lightbox = document.getElementById('lightbox');
    lightbox.classList.remove('active');
    document.removeEventListener('keydown', handleLightboxKeyboard);
}

function navigateLightbox(direction) {
    currentLightboxIndex += direction;
    if (currentLightboxIndex < 0) {
        currentLightboxIndex = allImages.length - 1;
    } else if (currentLightboxIndex >= allImages.length) {
        currentLightboxIndex = 0;
    }

    const img = document.getElementById('lightbox-img');
    img.src = allImages[currentLightboxIndex].source_url;
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

// Filter change handlers
document.getElementById('category-filter').addEventListener('change', () => {
    loadImages(1, false);
});

document.getElementById('device-filter').addEventListener('change', () => {
    loadImages(1, false);
});

// Close lightbox on background click
document.getElementById('lightbox').addEventListener('click', (e) => {
    if (e.target.id === 'lightbox') {
        closeLightbox();
    }
});

// Initialize
loadCategories();
loadImages();
