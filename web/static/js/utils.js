// API 工具函数
const API = {
    BASE: '/api/v1',

    async call(endpoint, data = {}, method = 'POST') {
        try {
            const options = {
                method: method,
                headers: { 'Content-Type': 'application/json' }
            };

            // 只在POST/PUT等请求中添加body
            if (method !== 'GET' && method !== 'HEAD') {
                options.body = JSON.stringify(data);
            }

            const response = await fetch(`${this.BASE}${endpoint}`, options);
            const result = await response.json();

            if (!response.ok) {
                throw new Error(result.error || 'Request failed');
            }

            return result;
        } catch (error) {
            console.error(`API call failed: ${endpoint}`, error);
            throw error;
        }
    },

    async get(endpoint, data = {}) {
        // 对于GET请求，将data作为查询参数
        let url = endpoint;
        if (Object.keys(data).length > 0) {
            const params = new URLSearchParams(data).toString();
            url = `${endpoint}?${params}`;
        }
        return this.call(url, {}, 'GET');
    },

    async post(endpoint, data = {}) {
        return this.call(endpoint, data, 'POST');
    }
};

// Modal 管理器
const ModalManager = {
    show(modalId, options = {}) {
        const modal = document.getElementById(modalId);
        if (!modal) {
            console.error(`Modal not found: ${modalId}`);
            return;
        }

        console.log(`Showing modal: ${modalId}`);
        const form = modal.querySelector('form');

        // 设置标题
        if (options.title) {
            const titleElement = modal.querySelector('.modal-title, [id$="-title"], [id$="-modal-title"]');
            if (titleElement) {
                titleElement.textContent = options.title;
            }
        }

        // 重置表单
        if (form && options.resetForm !== false) {
            form.reset();
        }

        // 设置默认值
        if (options.defaults) {
            Object.entries(options.defaults).forEach(([key, value]) => {
                const field = document.getElementById(key);
                if (field) {
                    if (field.type === 'checkbox') {
                        field.checked = value;
                    } else if (field.type === 'radio') {
                        if (field.value === String(value)) {
                            field.checked = true;
                        }
                    } else {
                        field.value = value || '';
                    }
                }
            });
        }

        // 显示前回调
        if (options.onShow) {
            options.onShow(modal);
        }

        modal.classList.add('active');
        console.log(`Modal ${modalId} is now active`);
    },

    hide(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('active');
            // Remove modal-open class when closing system config modal
            if (modalId === 'system-config-modal') {
                document.body.classList.remove('modal-open');
            }
        }
    },

    setupBackdropClick(modalId, closeCallback) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target.id === modalId) {
                    if (closeCallback) {
                        closeCallback();
                    } else {
                        this.hide(modalId);
                    }
                }
            });
        }
    }
};

// 通用删除处理器
async function deleteItem(endpoint, id, itemName, refreshCallback) {
    if (!confirm(`确定要删除这个${itemName}吗？`)) {
        return;
    }

    try {
        await API.post(endpoint, { id: id });
        showToast(`${itemName}已删除`, 'success');

        if (refreshCallback) {
            refreshCallback();
        }
    } catch (error) {
        console.error(`Failed to delete ${itemName}:`, error);
        showToast(`删除${itemName}失败`, 'error');
    }
}

// 通用表单提交处理器
async function submitForm(formConfig) {
    const {
        formId,
        idField,
        collectData,
        endpoints,
        itemName,
        closeModal,
        onSuccess
    } = formConfig;

    const id = document.getElementById(idField).value;
    const data = collectData();

    try {
        const endpoint = id ? endpoints.update : endpoints.add;
        const body = id ? { ...data, id: parseInt(id) } : data;

        await API.post(endpoint, body);

        showToast(id ? `${itemName}已更新` : `${itemName}已添加`, 'success');

        if (closeModal) {
            closeModal();
        }

        if (onSuccess) {
            onSuccess();
        }
    } catch (error) {
        console.error(`Failed to submit ${itemName}:`, error);
        showToast(`保存${itemName}失败`, 'error');
    }
}

// 通用编辑处理器
async function editItem(endpoint, id, formConfig) {
    try {
        const result = await API.post(endpoint, { id: id });
        const item = result.id !== undefined ? result : result[targetKey];

        const {
            modalId,
            titleId,
            titlePrefix,
            fieldMappings,
            onShow
        } = formConfig;

        // 设置标题
        if (titleId) {
            const titleElement = document.getElementById(titleId);
            if (titleElement) {
                titleElement.textContent = titlePrefix || '编辑';
            }
        }

        // 填充表单字段
        if (fieldMappings && item) {
            Object.entries(fieldMappings).forEach(([formField, itemField]) => {
                const value = typeof itemField === 'function' ? itemField(item) : item[itemField];
                const element = document.getElementById(formField);

                if (element) {
                    if (element.type === 'checkbox') {
                        element.checked = Boolean(value);
                    } else if (element.type === 'radio') {
                        if (element.value === String(value)) {
                            element.checked = true;
                        }
                    } else {
                        element.value = value !== null && value !== undefined ? String(value) : '';
                    }
                }
            });
        }

        // 显示前回调
        if (onShow) {
            await onShow(item);
        }

        // 显示模态框
        if (modalId) {
            ModalManager.show(modalId, { resetForm: false });
        }
    } catch (error) {
        console.error('Failed to load item:', error);
        showToast('加载详情失败', 'error');
    }
}

// 填充下拉选择框
function populateSelect(selectId, items, defaultText = '请选择', valueKey = 'id', labelKey = 'name') {
    const select = document.getElementById(selectId);
    if (!select) return;

    select.innerHTML = `<option value="">${defaultText}</option>`;

    items.forEach(item => {
        const option = document.createElement('option');
        option.value = item[valueKey];
        option.textContent = item[labelKey];
        select.appendChild(option);
    });
}

// 渲染空状态
function renderEmptyState(tbody, colSpan, message, icon = 'fa-inbox') {
    if (!tbody) return;

    tbody.innerHTML = `
        <tr>
            <td colspan="${colSpan}" style="text-align: center; padding: 40px; color: #6b7280;">
                <i class="fas ${icon}" style="font-size: 48px; margin-bottom: 10px;"></i>
                <p>${message}</p>
            </td>
        </tr>
    `;
}

// 格式化时间
function formatTime(timestamp) {
    if (!timestamp) return '-';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

// 格式化持续时间
function formatDuration(seconds) {
    if (!seconds) return '0秒';

    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    const parts = [];
    if (days > 0) parts.push(`${days}天`);
    if (hours > 0) parts.push(`${hours}小时`);
    if (minutes > 0) parts.push(`${minutes}分`);
    if (secs > 0 || parts.length === 0) parts.push(`${secs}秒`);

    return parts.join('');
}

// 防抖函数
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// 节流函数
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// 导出工具对象
window.API = API;
window.ModalManager = ModalManager;
window.deleteItem = deleteItem;
window.submitForm = submitForm;
window.editItem = editItem;
window.populateSelect = populateSelect;
window.renderEmptyState = renderEmptyState;
window.formatTime = formatTime;
window.formatDuration = formatDuration;
window.debounce = debounce;
window.throttle = throttle;
