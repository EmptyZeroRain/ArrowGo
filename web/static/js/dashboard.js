// Dashboard Page - Monitor List and Statistics

// Global state
let monitors = [];
let statuses = [];
let headersCount = 0;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadMonitors();
    loadStatuses();

    // Auto refresh every 30 seconds
    setInterval(refreshData, 30000);

    // Add first header row
    addHeaderRow();
});

// Load monitors
async function loadMonitors() {
    try {
        const data = await API.post('/monitor/list');
        monitors = data.targets || [];
        renderMonitors();
    } catch (error) {
        console.error('Failed to load monitors:', error);
        showToast('加载监控列表失败', 'error');
    }
}

// Load statuses
async function loadStatuses() {
    try {
        const data = await API.post('/monitor/status/list');
        statuses = data.statuses || [];
        updateStats();
        updateStatusInTable();
    } catch (error) {
        console.error('Failed to load statuses:', error);
    }
}

// Render monitors table
function renderMonitors() {
    const tbody = document.getElementById('monitors-tbody');

    if (monitors.length === 0) {
        renderEmptyState(tbody, 7, '暂无监控项，点击上方"添加监控"按钮开始添加');
        return;
    }

    tbody.innerHTML = monitors.map(monitor => {
        const status = statuses.find(s => s.target_id === monitor.id);
        const statusBadge = status ? getStatusBadge(status.status) : '<span class="status-badge unknown">未知</span>';
        const responseTime = status ? `${status.response_time}ms` : '-';
        const uptime = status ? `${status.uptime_percentage}%` : '-';

        return `
            <tr data-id="${monitor.id}">
                <td>
                    <strong>${monitor.name}</strong>
                    ${!monitor.enabled ? '<span style="color: #ef4444; font-size: 12px;">(已禁用)</span>' : ''}
                </td>
                <td>
                    <span style="text-transform: uppercase; font-weight: 600;">${monitor.type}</span>
                </td>
                <td>
                    <div style="display: flex; flex-direction: column;">
                        <span>${monitor.address}</span>
                        ${monitor.port ? `<span style="font-size: 12px; color: #6b7280;">:${monitor.port}</span>` : ''}
                    </div>
                </td>
                <td>${statusBadge}</td>
                <td>${responseTime}</td>
                <td>
                    <div style="display: flex; align-items: center; gap: 8px;">
                        <div style="flex: 1; height: 6px; background: #e5e7eb; border-radius: 3px; overflow: hidden;">
                            <div style="width: ${uptime}; height: 100%; background: ${getUptimeColor(status?.uptime_percentage || 0)}; border-radius: 3px;"></div>
                        </div>
                        <span style="font-size: 12px;">${uptime}</span>
                    </div>
                </td>
                <td>
                    <div style="display: flex; gap: 6px;">
                        <button class="btn btn-sm btn-secondary" onclick="editMonitor(${monitor.id})" title="编辑">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="deleteMonitor(${monitor.id})" title="删除">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

// Get status badge HTML
function getStatusBadge(status) {
    const badges = {
        'up': '<span class="status-badge up"><i class="fas fa-check-circle"></i> 在线</span>',
        'down': '<span class="status-badge down"><i class="fas fa-times-circle"></i> 离线</span>',
        'unknown': '<span class="status-badge unknown"><i class="fas fa-question-circle"></i> 未知</span>'
    };
    return badges[status] || badges['unknown'];
}

// Get uptime color
function getUptimeColor(uptime) {
    if (uptime >= 95) return '#10b981';
    if (uptime >= 80) return '#f59e0b';
    return '#ef4444';
}

// Update stats
function updateStats() {
    const upCount = statuses.filter(s => s.status === 'up').length;
    const downCount = statuses.filter(s => s.status === 'down').length;
    const totalCount = monitors.length;

    const avgResponseTime = statuses.length > 0
        ? Math.round(statuses.reduce((sum, s) => sum + s.response_time, 0) / statuses.length)
        : 0;

    document.getElementById('stat-up').textContent = upCount;
    document.getElementById('stat-down').textContent = downCount;
    document.getElementById('stat-total').textContent = totalCount;
    document.getElementById('stat-avg').textContent = `${avgResponseTime}ms`;
}

// Update status in table
function updateStatusInTable() {
    renderMonitors();
}

// Refresh data
function refreshData() {
    loadMonitors();
    loadStatuses();
    showToast('数据已刷新', 'success');
}

// Show add modal
function showAddModal() {
    ModalManager.show('monitor-modal', {
        title: '添加监控',
        defaults: {
            'monitor-id': '',
            'monitor-enabled': true,
            'monitor-interval': 60,
            'monitor-http-method': 'GET'
        },
        onShow: () => {
            updateFormFields();

            // Reset headers
            document.getElementById('headers-container').innerHTML = '';
            headersCount = 0;
            addHeaderRow();

            // Load DNS providers for the dropdown
            loadDNSProvidersForDropdown();
        }
    });
}

// Edit monitor
async function editMonitor(id) {
    try {
        const monitor = await API.post('/monitor/get', { id: id });

        // Parse headers
        let headers = {};
        if (monitor.http_headers) {
            try {
                headers = typeof monitor.http_headers === 'string'
                    ? JSON.parse(monitor.http_headers)
                    : monitor.http_headers;
            } catch (e) {
                console.error('Failed to parse headers:', e);
            }
        }

        ModalManager.show('monitor-modal', {
            title: '编辑监控',
            resetForm: false,
            defaults: {
                'monitor-id': monitor.id,
                'monitor-name': monitor.name,
                'monitor-type': monitor.type,
                'monitor-address': monitor.address,
                'monitor-port': monitor.port || '',
                'monitor-interval': monitor.interval || 60,
                'monitor-enabled': monitor.enabled,
                'monitor-http-method': monitor.http_method || 'GET',
                'monitor-http-body': monitor.http_body || '',
                'monitor-resolved-host': monitor.resolved_host || '',
                'monitor-dns-server': monitor.dns_server || '',
                'monitor-dns-server-name': monitor.dns_server_name || '',
                'monitor-dns-server-type': monitor.dns_server_type || 'udp',
                'monitor-snmp-community': monitor.snmp_community || '',
                'monitor-snmp-oid': monitor.snmp_oid || '',
                'monitor-snmp-version': monitor.snmp_version || 'v2c',
                'monitor-snmp-operator': monitor.snmp_operator || '',
                'monitor-snmp-expected-value': monitor.snmp_expected_value || ''
            },
            onShow: async () => {
                // Load headers
                document.getElementById('headers-container').innerHTML = '';
                headersCount = 0;
                Object.entries(headers).forEach(([key, value]) => {
                    addHeaderRow(key, value);
                });
                if (Object.keys(headers).length === 0) {
                    addHeaderRow();
                }

                updateFormFields();

                // Load DNS providers for the dropdown
                await loadDNSProvidersForDropdown();
            }
        });
    } catch (error) {
        console.error('Failed to load monitor:', error);
        showToast('加载监控详情失败', 'error');
    }
}

// Delete monitor
async function deleteMonitor(id) {
    await deleteItem('/monitor/remove', id, '监控', () => {
        loadMonitors();
        loadStatuses();
    });
}

// Close modal
function closeModal() {
    ModalManager.hide('monitor-modal');
}

// Update form fields based on type
function updateFormFields() {
    const type = document.getElementById('monitor-type').value;
    const httpSection = document.getElementById('http-section');
    const headersSection = document.getElementById('headers-section');
    const dnsSection = document.getElementById('dns-section');
    const snmpSection = document.getElementById('snmp-section');
    const portGroup = document.getElementById('port-group');

    // Hide all special sections first
    httpSection.style.display = 'none';
    headersSection.style.display = 'none';
    dnsSection.style.display = 'none';
    snmpSection.style.display = 'none';

    // Show/hide HTTP fields
    if (type === 'http' || type === 'https') {
        httpSection.style.display = 'block';
        headersSection.style.display = 'block';
        portGroup.style.display = 'block';
    }

    // Show/hide DNS fields
    if (type === 'dns') {
        dnsSection.style.display = 'block';
        portGroup.style.display = 'none';
    } else if (type === 'tcp' || type === 'udp') {
        dnsSection.style.display = 'block';
        portGroup.style.display = 'block';
    }

    // Show/hide SNMP fields
    if (type === 'snmp') {
        snmpSection.style.display = 'block';
        portGroup.style.display = 'block';
    }

    // Set default ports
    const portInput = document.getElementById('monitor-port');
    if (!portInput.value) {
        const defaultPorts = {
            'http': 80,
            'https': 443,
            'tcp': '',
            'udp': '',
            'snmp': 161
        };
        if (defaultPorts[type]) {
            portInput.value = defaultPorts[type];
        }
    }
}

// Add header row
function addHeaderRow(key = '', value = '') {
    headersCount++;
    const container = document.getElementById('headers-container');
    const row = document.createElement('div');
    row.className = 'header-row';
    row.innerHTML = `
        <div class="header-row-inputs">
            <input type="text" class="header-key" placeholder="Header 名称" value="${key}">
            <input type="text" class="header-value" placeholder="Header 值" value="${value}">
            <button type="button" class="btn btn-sm btn-danger" onclick="removeHeaderRow(this)">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `;
    container.appendChild(row);
}

// Remove header row
function removeHeaderRow(button) {
    button.closest('.header-row').remove();
}

// Collect headers from form
function collectHeaders() {
    const headers = {};
    const rows = document.querySelectorAll('.header-row');
    rows.forEach(row => {
        const key = row.querySelector('.header-key').value.trim();
        const value = row.querySelector('.header-value').value.trim();
        if (key && value) {
            headers[key] = value;
        }
    });
    return headers;
}

// Submit monitor form
async function submitMonitor(event) {
    event.preventDefault();

    const id = document.getElementById('monitor-id').value;
    const type = document.getElementById('monitor-type').value;

    const data = {
        name: document.getElementById('monitor-name').value,
        type: type,
        address: document.getElementById('monitor-address').value,
        port: parseInt(document.getElementById('monitor-port').value) || null,
        interval: parseInt(document.getElementById('monitor-interval').value) || 60,
        enabled: document.getElementById('monitor-enabled').checked
    };

    // HTTP/HTTPS specific fields
    if (type === 'http' || type === 'https') {
        data.http_method = document.getElementById('monitor-http-method').value;
        data.http_body = document.getElementById('monitor-http-body').value;
        data.resolved_host = document.getElementById('monitor-resolved-host').value;
        data.http_headers = collectHeaders();
    }

    // DNS specific fields
    if (type === 'dns' || type === 'http' || type === 'https') {
        data.dns_server = document.getElementById('monitor-dns-server').value;
        data.dns_server_name = document.getElementById('monitor-dns-server-name').value;
        data.dns_server_type = document.getElementById('monitor-dns-server-type').value;
    }

    // SNMP specific fields
    if (type === 'snmp') {
        data.snmp_community = document.getElementById('monitor-snmp-community').value;
        data.snmp_oid = document.getElementById('monitor-snmp-oid').value;
        data.snmp_version = document.getElementById('monitor-snmp-version').value;
        data.snmp_operator = document.getElementById('monitor-snmp-operator').value;
        data.snmp_expected_value = document.getElementById('monitor-snmp-expected-value').value;
    }

    try {
        const endpoint = id ? '/monitor/update' : '/monitor/add';
        const body = id ? { ...data, id: parseInt(id) } : data;

        await API.post(endpoint, body);

        showToast(id ? '监控已更新' : '监控已添加', 'success');
        closeModal();
        loadMonitors();
        loadStatuses();
    } catch (error) {
        console.error('Failed to submit monitor:', error);
        showToast('保存监控失败', 'error');
    }
}

// Query IP geolocation
async function queryIPGeo() {
    const ip = document.getElementById('ip-input').value.trim();
    if (!ip) {
        showToast('请输入 IP 地址', 'error');
        return;
    }

    try {
        const data = await API.post('/ipgeo/query', { ip: ip });

        document.getElementById('geo-ip').textContent = data.ip;
        document.getElementById('geo-country').textContent = data.country || '-';
        document.getElementById('geo-region').textContent = data.region || '-';
        document.getElementById('geo-city').textContent = data.city || '-';
        document.getElementById('geo-isp').textContent = data.isp || '-';
        document.getElementById('geo-coords').textContent =
            data.latitude && data.longitude ? `${data.latitude}, ${data.longitude}` : '-';

        document.getElementById('ipgeo-result').style.display = 'block';
    } catch (error) {
        console.error('Failed to query IP geolocation:', error);
        showToast('查询 IP 归属地失败', 'error');
    }
}

// Load DNS providers for dropdown in monitor form
async function loadDNSProvidersForDropdown() {
    try {
        const data = await API.post('/dns/provider/list');
        const providers = data.providers || [];

        const select = document.getElementById('monitor-dns-provider');
        select.innerHTML = '<option value="">手动输入</option>';

        providers.forEach(provider => {
            const option = document.createElement('option');
            option.value = provider.id;
            option.textContent = provider.name + (provider.is_default ? ' (默认)' : '');
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load DNS providers for dropdown:', error);
    }
}

// Select DNS provider and auto-fill fields
function selectDNSProvider() {
    const providerId = document.getElementById('monitor-dns-provider').value;

    if (!providerId) {
        return;
    }

    API.post('/dns/provider/list').then(data => {
        const providers = data.providers || [];
        const provider = providers.find(p => p.id == providerId);
        if (provider) {
            document.getElementById('monitor-dns-server-name').value = provider.name;
            document.getElementById('monitor-dns-server').value = provider.server;
            document.getElementById('monitor-dns-server-type').value = provider.server_type;
        }
    }).catch(error => {
        console.error('Failed to load DNS provider:', error);
    });
}

// Setup modal backdrop click
ModalManager.setupBackdropClick('monitor-modal', closeModal);

// Close modal on escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeModal();
    }
});
