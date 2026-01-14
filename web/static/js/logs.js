// Logs Page - Log Search

let currentLogPage = 0;
let totalLogs = 0;
let currentLogQuery = {};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadLogTargets();
});

// Load log targets
async function loadLogTargets() {
    try {
        const data = await API.post('/monitor/list');
        const monitors = data.targets || [];

        const select = document.getElementById('log-target');
        select.innerHTML = '<option value="">全部目标</option>';

        monitors.forEach(monitor => {
            const option = document.createElement('option');
            option.value = monitor.id;
            option.textContent = monitor.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load log targets:', error);
    }
}

// Search logs
async function searchLogs() {
    const targetId = document.getElementById('log-target').value;
    const status = document.getElementById('log-status').value;
    const queryText = document.getElementById('log-query').value;
    const size = parseInt(document.getElementById('log-size').value) || 20;

    currentLogQuery = {
        target_id: targetId ? parseInt(targetId) : null,
        status: status,
        query_text: queryText,
        size: size,
        from: 0
    };

    currentLogPage = 0;
    await loadLogs();
}

// Load logs
async function loadLogs(direction) {
    if (direction === 'prev' && currentLogPage > 0) {
        currentLogQuery.from -= currentLogQuery.size;
        currentLogPage--;
    } else if (direction === 'next') {
        currentLogQuery.from += currentLogQuery.size;
        currentLogPage++;
    }

    try {
        const data = await API.post('/logs/search', currentLogQuery);

        totalLogs = data.total || 0;
        renderLogs(data.hits || []);
        updateLogPagination();
    } catch (error) {
        if (error.message && error.message.includes('Elasticsearch is not enabled')) {
            document.getElementById('log-results').innerHTML = `
                <div style="text-align: center; padding: 40px; color: #6b7280;">
                    <i class="fas fa-exclamation-triangle" style="font-size: 48px; margin-bottom: 10px; color: #f59e0b;"></i>
                    <p>Elasticsearch 未启用</p>
                    <small>请在 config.yaml 中启用 Elasticsearch 以使用日志查询功能</small>
                </div>
            `;
            document.getElementById('log-pagination').style.display = 'none';
        } else {
            console.error('Failed to search logs:', error);
            showToast('搜索日志失败: ' + error.message, 'error');
        }
    }
}

// Render logs
function renderLogs(hits) {
    const container = document.getElementById('log-results');

    if (hits.length === 0) {
        container.innerHTML = `
            <div style="text-align: center; padding: 40px; color: #6b7280;">
                <i class="fas fa-search" style="font-size: 48px; margin-bottom: 10px;"></i>
                <p>未找到匹配的日志</p>
            </div>
        `;
        return;
    }

    container.innerHTML = hits.map(hit => {
        const statusBadgeClass = hit.status === 'up' ? 'log-up' : 'log-down';
        const timestamp = new Date(hit['@timestamp']).toLocaleString('zh-CN');

        return `
            <div class="log-entry" onclick="toggleLogEntry(this)">
                <div class="log-entry-header">
                    <div>
                        <strong>${hit.target_name}</strong>
                        <span class="status-badge ${statusBadgeClass}" style="margin-left: 10px;">
                            ${hit.status === 'up' ? '在线' : '离线'}
                        </span>
                    </div>
                    <div style="text-align: right;">
                        <div style="font-size: 14px; color: #6b7280;">${timestamp}</div>
                        <div style="font-size: 12px; color: #6b7280;">
                            响应时间: ${hit.response_time}ms
                        </div>
                    </div>
                </div>
                <div style="margin: 10px 0; font-size: 14px;">
                    ${hit.message}
                </div>
                <div class="log-entry-details">
                    <pre class="log-json">${JSON.stringify(hit, null, 2)}</pre>
                </div>
            </div>
        `;
    }).join('');
}

// Toggle log entry
function toggleLogEntry(element) {
    element.classList.toggle('expanded');
}

// Update log pagination
function updateLogPagination() {
    const pagination = document.getElementById('log-pagination');
    const pageInfo = document.getElementById('log-page-info');

    if (totalLogs === 0) {
        pagination.style.display = 'none';
        return;
    }

    pagination.style.display = 'block';
    const start = currentLogQuery.from + 1;
    const end = Math.min(start + currentLogQuery.size - 1, totalLogs);
    pageInfo.textContent = `显示 ${start}-${end} / 共 ${totalLogs} 条`;
}

// Close log modal
function closeLogModal() {
    ModalManager.hide('log-modal');
}

// Setup modal backdrop click
ModalManager.setupBackdropClick('log-modal', closeLogModal);

// Close modal on escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeLogModal();
    }
});
