// API utilities
(function() {
    const API_BASE = '/api/admin';

    // 获取或设置Admin Token
    function getAdminToken() {
        let token = localStorage.getItem('admin_token');
        if (!token) {
            token = prompt('请输入管理员Token:');
            if (token) {
                localStorage.setItem('admin_token', token.trim());
            }
        }
        return token;
    }

    // 清除Admin Token
    function clearAdminToken() {
        localStorage.removeItem('admin_token');
    }

    async function apiRequest(url, options = {}) {
        const token = getAdminToken();
        if (!token) {
            throw new Error('未设置管理员Token');
        }

        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            ...options.headers
        };

        const response = await fetch(API_BASE + url, {
            ...options,
            headers
        });

        if (!response.ok) {
            // 如果是401，清除token并提示重新输入
            if (response.status === 401) {
                clearAdminToken();
                throw new Error('Token无效，请刷新页面重新输入');
            }
            const error = await response.json();
            throw new Error(error.error || `HTTP ${response.status}`);
        }

        return await response.json();
    }

    // Alert system
    function showAlert(message, type = 'success') {
        const alert = document.createElement('div');
        alert.className = `alert alert-${type}`;
        alert.textContent = message;
        document.body.appendChild(alert);

        setTimeout(() => {
            alert.remove();
        }, 3000);
    }

    // Date formatter
    function formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleString('zh-CN');
    }

    // Export for use in other modules
    window.API = {
        request: apiRequest,
        BASE: API_BASE,
        getToken: getAdminToken,
        clearToken: clearAdminToken
    };

    window.Utils = {
        showAlert,
        formatDate
    };
})();
