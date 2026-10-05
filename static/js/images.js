// Image management module
const ImageManager = {
    currentPage: 1,
    selectedImages: new Set(),

    async loadImages(page = 1) {
        try {
            const data = await API.request(`/images?page=${page}&page_size=20`);
            this.renderImages(data.data);
            this.renderPagination('images-pagination', data.pagination);
            this.currentPage = page;
            this.updateSelection();
        } catch (error) {
            Utils.showAlert('加载图片失败: ' + error.message, 'error');
        }
    },

    renderImages(images) {
        const grid = document.getElementById('images-grid');
        grid.innerHTML = '';

        if (images.length === 0) {
            grid.innerHTML = '<p style="text-align: center; color: #666; padding: 40px;">暂无图片</p>';
            return;
        }

        images.forEach(image => {
            const card = this.createImageCard(image);
            grid.appendChild(card);
        });
    },

    createImageCard(image) {
        const card = document.createElement('div');
        card.className = 'image-card';
        card.innerHTML = `
            <input type="checkbox" data-id="${image.id}" onchange="ImageManager.toggleSelection(${image.id})">
            <img src="${image.source_url}" alt="${image.category?.name || ''}" loading="lazy">
            <div class="image-card-info">
                <p><strong>${image.category?.name || '未分类'}</strong></p>
                <p>${image.width || '?'} × ${image.height || '?'}</p>
                <p>${image.format || '未知格式'}</p>
            </div>
            <div class="image-card-actions">
                <button class="btn btn-sm btn-primary" onclick="ImageManager.editImage(${image.id})">编辑</button>
                <button class="btn btn-sm btn-danger" onclick="ImageManager.deleteImage(${image.id})">删除</button>
            </div>
        `;
        return card;
    },

    renderPagination(containerId, pagination) {
        const container = document.getElementById(containerId);
        container.innerHTML = '';

        const totalPages = pagination.total_page;
        const currentPage = pagination.page;

        if (totalPages <= 1) return;

        // Previous button
        if (currentPage > 1) {
            const prevBtn = document.createElement('button');
            prevBtn.textContent = '«';
            prevBtn.onclick = () => this.loadImages(currentPage - 1);
            container.appendChild(prevBtn);
        }

        // Page numbers
        const startPage = Math.max(1, currentPage - 2);
        const endPage = Math.min(totalPages, currentPage + 2);

        if (startPage > 1) {
            const btn = document.createElement('button');
            btn.textContent = '1';
            btn.onclick = () => this.loadImages(1);
            container.appendChild(btn);

            if (startPage > 2) {
                const dots = document.createElement('span');
                dots.textContent = '...';
                dots.style.padding = '0 10px';
                container.appendChild(dots);
            }
        }

        for (let i = startPage; i <= endPage; i++) {
            const btn = document.createElement('button');
            btn.textContent = i;
            btn.className = i === currentPage ? 'active' : '';
            btn.onclick = () => this.loadImages(i);
            container.appendChild(btn);
        }

        if (endPage < totalPages) {
            if (endPage < totalPages - 1) {
                const dots = document.createElement('span');
                dots.textContent = '...';
                dots.style.padding = '0 10px';
                container.appendChild(dots);
            }

            const btn = document.createElement('button');
            btn.textContent = totalPages;
            btn.onclick = () => this.loadImages(totalPages);
            container.appendChild(btn);
        }

        // Next button
        if (currentPage < totalPages) {
            const nextBtn = document.createElement('button');
            nextBtn.textContent = '»';
            nextBtn.onclick = () => this.loadImages(currentPage + 1);
            container.appendChild(nextBtn);
        }
    },

    toggleSelection(id) {
        if (this.selectedImages.has(id)) {
            this.selectedImages.delete(id);
        } else {
            this.selectedImages.add(id);
        }
        this.updateSelection();
    },

    updateSelection() {
        document.querySelectorAll('.image-card input[type="checkbox"]').forEach(checkbox => {
            const id = parseInt(checkbox.dataset.id);
            checkbox.checked = this.selectedImages.has(id);
        });
    },

    async deleteImage(id) {
        if (!confirm('确定要删除这张图片吗？')) return;

        try {
            await API.request(`/images/${id}`, { method: 'DELETE' });
            Utils.showAlert('删除成功');
            this.loadImages(this.currentPage);
        } catch (error) {
            Utils.showAlert('删除失败: ' + error.message, 'error');
        }
    },

    async batchDelete() {
        if (this.selectedImages.size === 0) {
            Utils.showAlert('请先选择图片', 'error');
            return;
        }

        if (!confirm(`确定要删除选中的 ${this.selectedImages.size} 张图片吗？`)) return;

        try {
            await API.request('/images/batch', {
                method: 'DELETE',
                body: JSON.stringify({ image_ids: Array.from(this.selectedImages) })
            });
            Utils.showAlert(`成功删除 ${this.selectedImages.size} 张图片`);
            this.selectedImages.clear();
            this.loadImages(this.currentPage);
        } catch (error) {
            Utils.showAlert('批量删除失败: ' + error.message, 'error');
        }
    }
};

window.ImageManager = ImageManager;
