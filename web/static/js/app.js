// Global state
let monitors = [];
let statuses = [];
let headersCount = 0;
let currentLogPage = 0;
let totalLogs = 0;
let currentLogQuery = {};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded, initializing application...');
    loadMonitors();
    loadStatuses();

    // Try to load system config on page load
    loadSystemConfig().catch(err => {
        console.warn('Initial config load failed (will retry when needed):', err.message);
    });

    // Auto refresh every 60 seconds (reduced frequency to improve performance)
    setInterval(refreshData, 60000);

    // Add first header row
    addHeaderRow();
});

// Load monitors
async function loadMonitors() {
    try {
        const data = await API.post('/monitor/list');
        monitors = data.targets || [];
        renderMonitors();
        populateLogTargets();
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

// Populate log targets dropdown
function populateLogTargets() {
    populateSelect('log-target', monitors, '全部目标', 'id', 'name');
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
                        <button class="btn btn-sm btn-secondary" onclick="viewMonitorDetails(${monitor.id})" title="详情">
                            <i class="fas fa-info-circle"></i>
                        </button>
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
    // Re-render monitors to show updated status
    renderMonitors();
}

// Refresh data
function refreshData() {
    loadMonitors();
    loadStatuses();
    // Removed toast notification to avoid annoyance during auto-refresh
}

// Switch tabs
function switchTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    event.target.closest('.tab-btn').classList.add('active');

    // Update tab contents
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    document.getElementById(`tab-${tabName}`).classList.add('active');
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
        onShow: async () => {
            updateFormFields();

            // Reset headers
            document.getElementById('headers-container').innerHTML = '';
            headersCount = 0;
            addHeaderRow();

            // Load DNS providers for the dropdown
            await loadDNSProvidersForDropdown();

            // Load alert channels for the dropdown
            await loadAlertChannelsForMonitor();

            // Reset SSL checkbox
            document.getElementById('monitor-ssl-check').checked = false;
            document.getElementById('ssl-config').style.display = 'none';

            // Add address input listener
            setupAddressInputListener();
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

        // Parse alert channels if exists
        let alertChannelIds = [];
        if (monitor.alert_channel_ids) {
            try {
                alertChannelIds = typeof monitor.alert_channel_ids === 'string'
                    ? JSON.parse(monitor.alert_channel_ids)
                    : monitor.alert_channel_ids;
            } catch (e) {
                console.error('Failed to parse alert channel ids:', e);
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
                'monitor-snmp-expected-value': monitor.snmp_expected_value || '',
                'monitor-ssl-check': monitor.ssl_check || false,
                'monitor-ssl-warn-days': monitor.ssl_warn_days || 30,
                'monitor-ssl-critical-days': monitor.ssl_critical_days || 7,
                'monitor-ssl-get-chain': monitor.ssl_get_chain !== false
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

                // Load alert channels for the dropdown
                await loadAlertChannelsForMonitor();

                // Set selected alert channels
                const alertChannelSelect = document.getElementById('monitor-alert-channels');
                Array.from(alertChannelSelect.options).forEach(option => {
                    option.selected = alertChannelIds.includes(parseInt(option.value));
                });

                // Toggle SSL options if needed
                if (monitor.ssl_check) {
                    document.getElementById('ssl-config').style.display = 'block';
                }

                // Add address input listener
                setupAddressInputListener();
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
    const sslOptions = document.getElementById('ssl-options');
    const addressInput = document.getElementById('monitor-address');

    // Hide all special sections first
    httpSection.style.display = 'none';
    headersSection.style.display = 'none';
    dnsSection.style.display = 'none';
    snmpSection.style.display = 'none';
    sslOptions.style.display = 'none';

    // Show/hide HTTP fields
    if (type === 'http' || type === 'https') {
        httpSection.style.display = 'block';
        headersSection.style.display = 'block';
        portGroup.style.display = 'block';

        // Show SSL options only for HTTPS
        if (type === 'https') {
            sslOptions.style.display = 'block';
        }
    }

    // Show/hide DNS fields based on address input
    checkAddressType();

    // Show/hide DNS fields for specific types
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
    const defaultPorts = {
        'http': 80,
        'https': 443,
        'tcp': '',
        'udp': '',
        'snmp': 161,
        'smtp': 25,
        'dns': 53
    };

    // Update port if:
    // 1. Port field is empty, OR
    // 2. Current port is a default port of another type
    const defaultPortValues = Object.values(defaultPorts).filter(p => p !== '');
    const shouldUpdateDefault = !portInput.value || defaultPortValues.includes(parseInt(portInput.value));

    if (shouldUpdateDefault && defaultPorts[type]) {
        portInput.value = defaultPorts[type];
    }
}

// Setup address input listener for auto-detection
function setupAddressInputListener() {
    const addressInput = document.getElementById('monitor-address');

    // Remove existing listener to avoid duplicates
    if (addressInput.dataset.listenerAttached) {
        return;
    }

    addressInput.addEventListener('blur', function() {
        parseAndUpdateFromAddress(this.value);
    });

    addressInput.addEventListener('change', function() {
        parseAndUpdateFromAddress(this.value);
    });

    addressInput.dataset.listenerAttached = 'true';
}

// Parse address and update form fields automatically
function parseAndUpdateFromAddress(address) {
    if (!address) return;

    const typeSelect = document.getElementById('monitor-type');
    const portInput = document.getElementById('monitor-port');
    const addressInput = document.getElementById('monitor-address');

    // Parse URL to extract protocol, host, port, and path
    let url = address.trim();

    // Check for protocol
    let protocol = '';
    if (url.startsWith('https://')) {
        protocol = 'https';
        url = url.substring(8); // Remove https://
    } else if (url.startsWith('http://')) {
        protocol = 'http';
        url = url.substring(7); // Remove http://
    }

    // Extract host, port, and path
    let host = url;
    let port = '';
    let path = '';

    const pathIndex = url.indexOf('/');
    if (pathIndex !== -1) {
        host = url.substring(0, pathIndex);
        path = url.substring(pathIndex); // Keep the / prefix
    }

    // Extract port from host:port
    const portIndex = host.lastIndexOf(':');
    const ipv6End = host.lastIndexOf(']');

    // Check if it's an IPv6 address
    if (host.indexOf('[') === 0 && ipv6End !== -1) {
        // IPv6 format: [2001:db8::1]:port or [2001:db8::1]
        const ipv6Host = host.substring(0, ipv6End + 1);
        if (portIndex > ipv6End) {
            port = host.substring(portIndex + 1);
            host = ipv6Host;
        } else {
            host = ipv6Host;
        }
    } else if (portIndex !== -1 && !host.match(/^[\d.]+$/)) {
        // Not an IP address with port, so it's hostname:port
        const parts = host.split(':');
        host = parts[0];
        port = parts[1];
    } else if (host.match(/^[\d.]+:\d+$/)) {
        // IPv4 with port: 192.168.1.1:8080
        const parts = host.split(':');
        host = parts[0];
        port = parts[1];
    }

    // Store the full URL in data attribute for submission
    const fullUrl = (protocol ? protocol + '://' : '') + addressInput.value;

    // Update type based on protocol
    if (protocol) {
        typeSelect.value = protocol;
        // Trigger form update
        updateFormFields();
    }

    // Update port if found and not already set to a custom value
    if (port && !isCustomPort(portInput.value)) {
        portInput.value = port;
    } else if (!port && protocol && !isCustomPort(portInput.value)) {
        // Set default port based on protocol
        if (protocol === 'https') {
            portInput.value = '443';
        } else if (protocol === 'http') {
            portInput.value = '80';
        }
    }

    // Keep the full URL in the address field (don't strip path)
    // But store the hostname for DNS checking
    addressInput.dataset.hostname = host;

    // Check address type for DNS section (using extracted hostname)
    checkAddressTypeForHost(host);
}

// Check if port is a custom (non-default) port
function isCustomPort(portValue) {
    if (!portValue) return false;
    const port = parseInt(portValue);
    const defaultPorts = [80, 443, 8080, 3000, 5000, 8000, 8888];
    return !defaultPorts.includes(port);
}

// Check if address is a domain or IP (using extracted hostname)
function checkAddressType() {
    const addressInput = document.getElementById('monitor-address');
    const hostname = addressInput.dataset.hostname || addressInput.value.trim();
    checkAddressTypeForHost(hostname);
}

// Check address type for a specific hostname
function checkAddressTypeForHost(hostname) {
    const dnsSection = document.getElementById('dns-section');
    const type = document.getElementById('monitor-type').value;
    const dnsProviderSelect = document.getElementById('monitor-dns-provider');
    const resolvedHostInput = document.getElementById('monitor-resolved-host');

    // Only show DNS section for HTTP/HTTPS if address looks like a domain
    if ((type === 'http' || type === 'https') && hostname) {
        // Check if it's a domain (not an IP)
        const isDomain = !/^(\d+\.){3}\d+$/.test(hostname) && !hostname.includes('[');
        if (isDomain) {
            dnsSection.style.display = 'block';

            // If DNS provider is already selected, keep custom host disabled
            if (dnsProviderSelect.value) {
                resolvedHostInput.disabled = true;
                resolvedHostInput.placeholder = "已选择DNS供应商，无需手动配置";
            }
        } else {
            // For IP addresses, hide DNS section and enable custom host
            dnsSection.style.display = 'none';
            resolvedHostInput.disabled = false;
            resolvedHostInput.placeholder = "例如: example.com";
        }
    }
}

// Toggle SSL options visibility
function toggleSSLOptions() {
    const sslCheck = document.getElementById('monitor-ssl-check');
    const sslConfig = document.getElementById('ssl-config');

    if (sslCheck.checked) {
        sslConfig.style.display = 'block';
    } else {
        sslConfig.style.display = 'none';
    }
}

// Add header row
// Common HTTP headers presets
const commonHeaders = [
    { key: 'User-Agent', value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36' },
    { key: 'Accept', value: 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' },
    { key: 'Accept-Language', value: 'zh-CN,zh;q=0.9,en;q=0.8' },
    { key: 'Accept-Encoding', value: 'gzip, deflate, br' },
    { key: 'Connection', value: 'keep-alive' },
    { key: 'Cache-Control', value: 'max-age=0' },
    { key: 'Content-Type', value: 'application/json' }
];

function addHeaderRow(key = '', value = '') {
    headersCount++;
    const container = document.getElementById('headers-container');
    const row = document.createElement('div');
    row.className = 'header-row';

    // Create options for common headers
    const headerOptions = commonHeaders.map(h =>
        `<option value="${h.key}">${h.key}</option>`
    ).join('');

    row.innerHTML = `
        <div class="header-row-inputs">
            <select class="header-key-select" onchange="if(this.value==='custom'){this.nextElementSibling.style.display='block';this.style.display='none';}else{this.nextElementSibling.value=this.value;this.nextElementSibling.style.display='none';this.style.display='block';}">
                <option value="">选择常见Header...</option>
                ${headerOptions}
                <option value="custom">自定义...</option>
            </select>
            <input type="text" class="header-key" placeholder="Header 名称" value="${key}" style="display: ${key ? 'block' : 'none'};">
            <input type="text" class="header-value" placeholder="Header 值" value="${value}">
            <button type="button" class="btn btn-sm btn-secondary" onclick="fillHeaderValue(this)" title="填充预设值">
                <i class="fas fa-magic"></i>
            </button>
            <button type="button" class="btn btn-sm btn-danger" onclick="removeHeaderRow(this)">
                <i class="fas fa-times"></i>
            </button>
        </div>
    `;
    container.appendChild(row);
}

// Fill header value based on key
function fillHeaderValue(button) {
    const row = button.closest('.header-row');
    const keyInput = row.querySelector('.header-key');
    const valueInput = row.querySelector('.header-value');
    const key = keyInput.value.trim();

    const header = commonHeaders.find(h => h.key === key);
    if (header) {
        valueInput.value = header.value;
    }
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
    const addressInput = document.getElementById('monitor-address');
    let address = addressInput.value.trim();

    // For HTTP/HTTPS, keep the full URL including path
    // For other types, use the address as-is
    const data = {
        name: document.getElementById('monitor-name').value,
        type: type,
        address: address,
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

        // SSL/TLS specific fields (only for HTTPS)
        if (type === 'https') {
            data.ssl_check = document.getElementById('monitor-ssl-check').checked;
            if (data.ssl_check) {
                data.ssl_warn_days = parseInt(document.getElementById('monitor-ssl-warn-days').value) || 30;
                data.ssl_critical_days = parseInt(document.getElementById('monitor-ssl-critical-days').value) || 7;
                data.ssl_get_chain = document.getElementById('monitor-ssl-get-chain').checked;
            }
        }
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

    // Alert channels
    const alertChannelSelect = document.getElementById('monitor-alert-channels');
    const selectedChannels = Array.from(alertChannelSelect.selectedOptions).map(option => parseInt(option.value));
    data.alert_channel_ids = selectedChannels;

    try {
        const endpoint = id ? '/monitor/update' : '/monitor/add';
        const body = id ? { ...data, id: parseInt(id) } : data;

        await API.post(endpoint, body);

        // Show success message with longer duration for create/update actions
        const message = id ? '监控已更新' : '监控已创建成功';
        const toast = document.getElementById('toast');
        toast.textContent = message;
        toast.className = 'toast success active';
        setTimeout(() => {
            toast.classList.remove('active');
        }, 5000); // Show for 5 seconds instead of 3

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

        // Populate geo-info div dynamically
        const geoInfo = document.getElementById('geo-info');
        const fields = [
            { label: 'IP 地址', value: data.ip },
            { label: '国家', value: data.country },
            { label: '地区', value: data.region },
            { label: '城市', value: data.city },
            { label: 'ISP', value: data.isp },
            { label: '坐标', value: data.latitude && data.longitude ? `${data.latitude}, ${data.longitude}` : '' }
        ];

        geoInfo.innerHTML = '';
        fields.forEach(field => {
            const item = document.createElement('div');
            item.style.cssText = 'display: flex; justify-content: space-between; padding: var(--spacing-2) 0; border-bottom: 1px solid var(--color-gray-200);';
            item.innerHTML = `
                <label style="font-weight: var(--font-semibold); color: var(--color-gray-700);">${field.label}:</label>
                <span style="color: var(--color-gray-900);">${field.value || '-'}</span>
            `;
            geoInfo.appendChild(item);
        });

        document.getElementById('ipgeo-result').style.display = 'block';
    } catch (error) {
        console.error('Failed to query IP geolocation:', error);
        showToast('查询 IP 归属地失败', 'error');
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

// Show log detail
function showLogDetail(logId) {
    // Implement log detail view if needed
    console.log('Show log detail:', logId);
}

// Close log modal
function closeLogModal() {
    ModalManager.hide('log-modal');
}

// View monitor details
async function viewMonitorDetails(id) {
    try {
        // Load monitor info
        const monitor = await API.post('/monitor/get', { id: id });

        // Load latest status
        const statusData = await API.post('/monitor/status/list', {
            target_id: id,
            limit: 1
        });

        // Load recent check logs (last 20)
        const logsData = await API.post('/monitor/status/list', {
            target_id: id,
            limit: 20
        });

        const status = statusData.statuses && statusData.statuses.length > 0 ? statusData.statuses[0] : null;
        const logs = logsData.statuses || [];

        // Debug: log the status object
        console.log('Status object:', status);
        console.log('Status.data type:', typeof status?.data);
        console.log('Status.data value:', status?.data);

        const content = document.getElementById('monitor-details-content');
        let html = `
            <div style="padding: var(--spacing-6);">
                <div class="form-section">
                    <h3>基本信息</h3>
                    <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                        <p><strong>名称:</strong> ${monitor.name}</p>
                        <p><strong>类型:</strong> ${monitor.type.toUpperCase()}</p>
                        <p><strong>地址:</strong> ${monitor.address}${monitor.port ? ':' + monitor.port : ''}</p>
                        <p><strong>检查间隔:</strong> ${monitor.interval}秒</p>
                        <p><strong>状态:</strong> ${monitor.enabled ? '<span style="color: var(--color-success-600);">启用</span>' : '<span style="color: var(--color-gray-500);">禁用</span>'}</p>
                        <p><strong>创建时间:</strong> ${new Date(monitor.created_at).toLocaleString('zh-CN')}</p>
                    </div>
                </div>
        `;

        if (status) {
            const statusBadge = getStatusBadge(status.status);

            // Add domain details section for HTTP/HTTPS monitors
            if (monitor.type === 'http' || monitor.type === 'https') {
                // Extract hostname from address
                let hostname = monitor.address;
                if (hostname.startsWith('http://') || hostname.startsWith('https://')) {
                    try {
                        const url = new URL(hostname);
                        hostname = url.hostname;
                    } catch (e) {
                        // If URL parsing fails, use the address as-is
                    }
                } else if (hostname.includes(':')) {
                    hostname = hostname.split(':')[0];
                } else if (hostname.includes('/')) {
                    hostname = hostname.split('/')[0];
                }

                html += `
                    <div class="form-section">
                        <h3><i class="fas fa-globe"></i> 域名详情</h3>
                        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                            <p><strong>域名:</strong> <code style="background: var(--color-primary-50); padding: 2px 6px; border-radius: 4px;">${hostname}</code></p>
                            <p><strong>IP地址:</strong> <code style="background: var(--color-success-50); padding: 2px 6px; border-radius: 4px; color: var(--color-success-700);">${status.resolved_ip || '未解析'}</code></p>
                        </div>
                        <div id="ip-geo-info-${monitor.id}" style="margin-top: var(--spacing-2); font-size: 13px; color: var(--color-gray-600);">
                            <i class="fas fa-spinner fa-spin"></i> 正在查询IP归属地...
                        </div>
                    </div>
                `;
            }

            html += `
                <div class="form-section">
                    <h3>当前状态</h3>
                    <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                        <p><strong>状态:</strong> ${statusBadge}</p>
                        <p><strong>响应时间:</strong> ${status.response_time}ms</p>
                        <p><strong>检查时间:</strong> ${new Date(status.checked_at).toLocaleString('zh-CN')}</p>
                        <p><strong>正常运行时间:</strong> ${status.uptime_percentage}%</p>
                    </div>
                    ${monitor.type !== 'https' ? `<p style="margin-top: var(--spacing-3);"><strong>消息:</strong> ${status.message}</p>` : ''}
                </div>
            `;

            // Parse data field if exists
            let statusData = {};
            if (status.data) {
                try {
                    if (typeof status.data === 'string') {
                        statusData = JSON.parse(status.data);
                        console.log('Parsed status.data successfully:', statusData);
                    } else {
                        statusData = status.data;
                        console.log('status.data is already an object:', statusData);
                    }
                } catch (e) {
                    console.error('Failed to parse status data:', e);
                    console.error('Raw status.data:', status.data);
                }
            }

            // Show SSL certificate chain if available (for HTTPS type)
            if (monitor.type === 'https') {
                console.log('HTTPS monitor detected, statusData:', statusData);
                const chain = statusData.certificate_chain;
                console.log('Certificate chain:', chain);

                // Always show SSL certificate section for HTTPS monitors
                html += `
                    <div class="form-section">
                        <h3><i class="fas fa-lock"></i> SSL/TLS 证书信息</h3>
                `;

                // If no chain in data but has basic SSL info, show basic info
                if (!chain && (status.ssl_issuer || status.ssl_subject)) {
                    html += `
                        <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                            ${status.ssl_subject ? `<p><strong>主题:</strong> ${status.ssl_subject}</p>` : ''}
                            ${status.ssl_issuer ? `<p><strong>颁发者:</strong> ${status.ssl_issuer}</p>` : ''}
                            ${status.ssl_serial ? `<p><strong>序列号:</strong> <code style="font-size: 11px;">${status.ssl_serial.substring(0, 32)}...</code></p>` : ''}
                            ${status.ssl_days_until_expiry !== undefined && status.ssl_days_until_expiry !== null ? `<p><strong>剩余天数:</strong> <span style="color: ${status.ssl_days_until_expiry < 30 ? 'var(--color-warning-600);' : 'var(--color-success-600);'}; font-weight: bold;">${status.ssl_days_until_expiry} 天</span></p>` : ''}
                        </div>
                    `;
                }

                // Show full certificate chain if available
                if (chain && Array.isArray(chain) && chain.length > 0) {
                    html += `
                        <h4 style="margin-top: var(--spacing-4); margin-bottom: var(--spacing-3);">证书链详情</h4>
                        <div style="display: flex; flex-direction: column; gap: var(--spacing-4);">
                    `;

                    chain.forEach((cert, index) => {
                        const expiryClass = cert.days_until_expiry < 7 ? 'color: var(--color-danger-600);' :
                                           cert.days_until_expiry < 30 ? 'color: var(--color-warning-600);' :
                                           'color: var(--color-success-600);';

                        const role = cert.is_ca ? (index === chain.length - 1 ? '根证书' : '中间证书') : '终端证书';

                        html += `
                            <div style="background: var(--color-gray-50); border: 1px solid var(--color-gray-200); border-radius: var(--radius-md); padding: var(--spacing-4);">
                                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--spacing-3);">
                                    <h4 style="margin: 0; color: var(--color-primary-600);">
                                        <span style="background: var(--color-primary-100); padding: 2px 8px; border-radius: 4px; font-size: 12px;">
                                            ${index + 1}
                                        </span>
                                        ${cert.subject_cn}
                                    </h4>
                                    <span style="background: var(--color-gray-200); padding: 4px 8px; border-radius: 4px; font-size: 12px;">
                                        ${role}
                                    </span>
                                </div>
                                <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-2); font-size: 13px;">
                                    <div><strong>颁发者:</strong> ${cert.issuer_cn}</div>
                                    <div><strong>序列号:</strong> <code style="font-size: 11px;">${cert.serial.substring(0, 16)}...</code></div>
                                    <div><strong>生效日期:</strong> ${new Date(cert.not_before).toLocaleDateString('zh-CN')}</div>
                                    <div><strong>过期日期:</strong> ${new Date(cert.not_after).toLocaleDateString('zh-CN')}</div>
                                    <div style="grid-column: span 2;"><strong>剩余天数:</strong> <span style="${expiryClass} font-weight: bold;">${cert.days_until_expiry} 天</span></div>
                                    ${cert.subject_org ? `<div><strong>组织:</strong> ${cert.subject_org}</div>` : ''}
                                    ${cert.signature_algorithm ? `<div><strong>签名算法:</strong> ${cert.signature_algorithm}</div>` : ''}
                                    ${cert.dns_names && cert.dns_names.length > 0 ? `<div style="grid-column: span 2;"><strong>DNS名称:</strong> ${cert.dns_names.join(', ')}</div>` : ''}
                                </div>
                            </div>
                        `;
                    });

                    html += `
                            </div>
                    `;
                }

                html += `
                    </div>
                `;
            }
        }

        // Show HTTP details if available
        if (monitor.type === 'http' || monitor.type === 'https') {
            html += `
                <div class="form-section">
                    <h3>HTTP 配置</h3>
                    <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                        <p><strong>方法:</strong> ${monitor.http_method || 'GET'}</p>
                        ${monitor.resolved_host ? `<p><strong>自定义Host:</strong> ${monitor.resolved_host}</p>` : ''}
                        ${monitor.dns_server_name ? `<p><strong>DNS供应商:</strong> ${monitor.dns_server_name} (${monitor.dns_server})</p>` : ''}
                    </div>
                    ${monitor.http_headers ? `<p style="margin-top: var(--spacing-2);"><strong>请求头:</strong> <code style="font-size: 12px;">${Object.keys(JSON.parse(monitor.http_headers || '{}')).join(', ')}</code></p>` : ''}
                </div>
            `;
        }

        // Show alert channels
        if (monitor.alert_channel_ids) {
            let channelIds = [];
            try {
                channelIds = typeof monitor.alert_channel_ids === 'string'
                    ? JSON.parse(monitor.alert_channel_ids)
                    : monitor.alert_channel_ids;
            } catch (e) {}

            if (channelIds.length > 0) {
                html += `
                    <div class="form-section">
                        <h3>关联告警通道</h3>
                        <p><strong>已关联 ${channelIds.length} 个告警通道</strong></p>
                    </div>
                `;
            }
        }

        // Show check logs
        html += `
            <div class="form-section">
                <h3><i class="fas fa-history"></i> 最近拨测日志 (最近20条)</h3>
        `;

        if (logs.length === 0) {
            html += `<p style="text-align: center; color: var(--color-gray-500); padding: var(--spacing-4);">暂无拨测日志</p>`;
        } else {
            html += `
                <div class="table-container" style="margin-top: var(--spacing-4);">
                    <table class="table">
                        <thead>
                            <tr>
                                <th>时间</th>
                                <th>状态</th>
                                <th>响应时间</th>
                                <th>消息</th>
                                <th>操作</th>
                            </tr>
                        </thead>
                        <tbody>
            `;

            logs.forEach(log => {
                const statusClass = log.status === 'up' ? 'success' : log.status === 'down' ? 'danger' : 'warning';
                html += `
                    <tr>
                        <td style="font-size: 12px;">${new Date(log.checked_at).toLocaleString('zh-CN')}</td>
                        <td><span class="status-badge ${log.status}">${log.status.toUpperCase()}</span></td>
                        <td>${log.response_time}ms</td>
                        <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${log.message}">${log.message}</td>
                        <td>
                            <button class="btn btn-sm btn-secondary" onclick='showCheckLogDetail(${JSON.stringify(log)})' title="查看详情">
                                <i class="fas fa-eye"></i>
                            </button>
                        </td>
                    </tr>
                `;
            });

            html += `
                        </tbody>
                    </table>
                </div>
            `;
        }

        html += `
                </div>
            </div>
        `;

        content.innerHTML = html;

        // Fetch IP geolocation info if IP is resolved
        if (status && status.resolved_ip && (monitor.type === 'http' || monitor.type === 'https')) {
            // Only fetch geo info if resolved_ip looks like an IP address (not a domain)
            if (/^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$/.test(status.resolved_ip) || /^(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$/.test(status.resolved_ip)) {
                fetchIPGeoLocation(status.resolved_ip, monitor.id);
            } else {
                // If resolved_ip is a domain, hide the geo info section
                const geoInfoDiv = document.getElementById(`ip-geo-info-${monitor.id}`);
                if (geoInfoDiv) {
                    geoInfoDiv.style.display = 'none';
                }
            }
        }

        ModalManager.show('monitor-details-modal', {
            title: `监控详情 - ${monitor.name}`,
            resetForm: false
        });
    } catch (error) {
        console.error('Failed to load monitor details:', error);
        showToast('加载监控详情失败', 'error');
    }
}

// Fetch IP geolocation information
async function fetchIPGeoLocation(ip, monitorId) {
    const geoInfoDiv = document.getElementById(`ip-geo-info-${monitorId}`);
    if (!geoInfoDiv) return;

    try {
        // Get geo info from local database
        const response = await fetch(`/api/v1/ip/geo/${ip}`);
        if (response.ok) {
            const geoData = await response.json();

            // Check if we have valid geo data
            if (geoData && (geoData.country || geoData.region || geoData.city || geoData.isp)) {
                const geoText = [
                    geoData.country || '',
                    geoData.region || '',
                    geoData.city || '',
                    geoData.isp ? `(${geoData.isp})` : ''
                ].filter(Boolean).join(' ');

                geoInfoDiv.innerHTML = `
                    <i class="fas fa-map-marker-alt" style="color: var(--color-success-500);"></i>
                    归属地信息: ${geoText || '未知'}
                `;
            } else {
                geoInfoDiv.innerHTML = `
                    <i class="fas fa-info-circle" style="color: var(--color-info-500);"></i>
                    归属地信息: 暂无数据
                `;
            }
        } else {
            geoInfoDiv.innerHTML = `
                <i class="fas fa-exclamation-circle" style="color: var(--color-warning-500);"></i>
                归属地信息: 暂无数据
            `;
        }
    } catch (error) {
        console.error('Failed to fetch IP geo location:', error);
        geoInfoDiv.innerHTML = `
            <i class="fas fa-exclamation-circle" style="color: var(--color-warning-500);"></i>
            归属地信息: 查询失败
        `;
    }
}

// Show check log detail
function showCheckLogDetail(log) {
    const content = document.getElementById('check-log-detail-content');

    let html = `
        <div style="padding: var(--spacing-6);">
            <div class="form-section">
                <h3>基本信息</h3>
                <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                    <p><strong>检查时间:</strong> ${new Date(log.checked_at).toLocaleString('zh-CN')}</p>
                    <p><strong>状态:</strong> ${getStatusBadge(log.status)}</p>
                    <p><strong>响应时间:</strong> ${log.response_time}ms</p>
                    <p><strong>正常运行时间:</strong> ${log.uptime_percentage}%</p>
                </div>
            </div>

            <div class="form-section">
                <h3>检查消息</h3>
                <div style="background: var(--color-gray-50); padding: var(--spacing-4); border-radius: var(--radius-md); white-space: pre-wrap; word-wrap: break-word;">${log.message}</div>
            </div>
    `;

    // Show request details if available
    if (log.request) {
        html += `
            <div class="form-section">
                <h3>请求详情</h3>
                <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                    <p><strong>方法:</strong> ${log.request.method || '-'}</p>
                    <p><strong>URL:</strong> ${log.request.url || '-'}</p>
                </div>
        `;

        // Show request headers if available
        if (log.request.headers && Object.keys(log.request.headers).length > 0) {
            html += `
                <div style="margin-top: var(--spacing-4);">
                    <h4 style="margin-bottom: var(--spacing-2);">请求头:</h4>
                    <div style="background: var(--color-gray-50); padding: var(--spacing-4); border-radius: var(--radius-md); font-size: 12px;">
            `;
            Object.entries(log.request.headers).forEach(([key, value]) => {
                html += `<p style="margin: var(--spacing-1) 0;"><strong>${key}:</strong> ${value}</p>`;
            });
            html += `
                    </div>
                </div>
            `;
        }

        // Show request body if available
        if (log.request.body) {
            html += `
                <div style="margin-top: var(--spacing-4);">
                    <h4 style="margin-bottom: var(--spacing-2);">请求体:</h4>
                    <div style="background: var(--color-gray-50); padding: var(--spacing-4); border-radius: var(--radius-md); font-size: 12px; white-space: pre-wrap; word-wrap: break-word;">${log.request.body}</div>
                </div>
            `;
        }

        html += `</div>`;
    }

    // Show response details if available
    if (log.response) {
        html += `
            <div class="form-section">
                <h3>响应详情</h3>
                <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--spacing-3);">
                    <p><strong>状态码:</strong> ${log.response.status_code !== undefined ? log.response.status_code : '-'}</p>
                    <p><strong>大小:</strong> ${log.response.body_size || log.response.size || '-'} bytes</p>
                </div>
        `;

        // Show response headers if available
        if (log.response.headers && Object.keys(log.response.headers).length > 0) {
            html += `
                <div style="margin-top: var(--spacing-4);">
                    <h4 style="margin-bottom: var(--spacing-2);">响应头:</h4>
                    <div style="background: var(--color-gray-50); padding: var(--spacing-4); border-radius: var(--radius-md); font-size: 12px;">
            `;
            Object.entries(log.response.headers).forEach(([key, value]) => {
                html += `<p style="margin: var(--spacing-1) 0;"><strong>${key}:</strong> ${value}</p>`;
            });
            html += `
                    </div>
                </div>
            `;
        }

        html += `</div>`;
    }

    // Show certificate chain if available
    if (log.data && log.data.certificate_chain) {
        const chain = log.data.certificate_chain;
        html += `
            <div class="form-section">
                <h3><i class="fas fa-lock"></i> SSL/TLS 证书链</h3>
                <div style="display: flex; flex-direction: column; gap: var(--spacing-3);">
        `;

        chain.forEach((cert, index) => {
            const role = cert.is_ca ? (index === chain.length - 1 ? '根证书' : '中间证书') : '终端证书';
            html += `
                <div style="background: var(--color-gray-50); border: 1px solid var(--color-gray-200); border-radius: var(--radius-md); padding: var(--spacing-3);">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--spacing-2);">
                        <h4 style="margin: 0; font-size: 14px;">${index + 1}. ${cert.subject_cn} <span style="font-size: 12px; color: var(--color-gray-600);">(${role})</span></h4>
                        <span style="font-size: 12px; color: ${cert.days_until_expiry < 30 ? 'var(--color-warning-600);' : 'var(--color-success-600);'}">${cert.days_until_expiry} 天后过期</span>
                    </div>
                    <div style="font-size: 12px; color: var(--color-gray-600);">
                        <div>颁发者: ${cert.issuer_cn}</div>
                        <div>过期: ${new Date(cert.not_after).toLocaleDateString('zh-CN')}</div>
                    </div>
                </div>
            `;
        });

        html += `
                </div>
            </div>
        `;
    }

    // Show error details if available
    if (log.error) {
        html += `
            <div class="form-section">
                <h3>错误详情</h3>
                <div style="background: var(--color-danger-50); padding: var(--spacing-4); border-radius: var(--radius-md); color: var(--color-danger-600);">${log.error}</div>
            </div>
        `;
    }

    html += `</div>`;

    content.innerHTML = html;

    ModalManager.show('check-log-detail-modal', {
        title: '拨测日志详情',
        resetForm: false
    });
}

// Close monitor details modal
function closeMonitorDetailsModal() {
    ModalManager.hide('monitor-details-modal');
}

// Close check log detail modal
function closeCheckLogDetailModal() {
    ModalManager.hide('check-log-detail-modal');
}

// Show toast notification
function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast ${type} active`;

    setTimeout(() => {
        toast.classList.remove('active');
    }, 3000);
}

// Close modal on escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeModal();
        closeLogModal();
        closeMonitorDetailsModal();
        closeCheckLogDetailModal();
        closeDNSProviderModal();
        closeAlertChannelModal();
        closeAlertRuleModal();
        closeSystemConfigModal();
    }
});

// Setup modal backdrop clicks
ModalManager.setupBackdropClick('monitor-modal', closeModal);
ModalManager.setupBackdropClick('log-modal', closeLogModal);
ModalManager.setupBackdropClick('monitor-details-modal', closeMonitorDetailsModal);
ModalManager.setupBackdropClick('check-log-detail-modal', closeCheckLogDetailModal);
ModalManager.setupBackdropClick('dns-provider-modal', closeDNSProviderModal);
ModalManager.setupBackdropClick('alert-channel-modal', closeAlertChannelModal);
ModalManager.setupBackdropClick('alert-rule-modal', closeAlertRuleModal);
ModalManager.setupBackdropClick('system-config-modal', closeSystemConfigModal);

// DNS Provider Management

let dnsProviders = [];

// Load DNS providers
async function loadDNSProviders() {
    try {
        const data = await API.post('/dns/provider/list');
        dnsProviders = data.providers || [];
        renderDNSProviders();
    } catch (error) {
        console.error('Failed to load DNS providers:', error);
        showToast('加载DNS供应商列表失败', 'error');
    }
}

// Render DNS providers table
function renderDNSProviders() {
    const tbody = document.getElementById('dns-providers-tbody');

    if (dnsProviders.length === 0) {
        renderEmptyState(tbody, 5, '暂无DNS供应商，点击上方"添加DNS供应商"按钮开始添加');
        return;
    }

    tbody.innerHTML = dnsProviders.map(provider => {
        const typeLabels = {
            'udp': 'UDP',
            'tcp': 'TCP',
            'doh': 'DoH',
            'dot': 'DoT'
        };
        const typeLabel = typeLabels[provider.server_type] || provider.server_type.toUpperCase();
        const defaultBadge = provider.is_default
            ? '<span style="color: #10b981; font-weight: 600;"><i class="fas fa-star"></i> 默认</span>'
            : '';

        return `
            <tr data-id="${provider.id}">
                <td><strong>${provider.name}</strong></td>
                <td><code style="background: #f3f4f6; padding: 2px 6px; border-radius: 3px;">${provider.server}</code></td>
                <td>${typeLabel}</td>
                <td>${defaultBadge}</td>
                <td>
                    <div style="display: flex; gap: 6px;">
                        <button class="btn btn-sm btn-secondary" onclick="editDNSProvider(${provider.id})" title="编辑">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="deleteDNSProvider(${provider.id})" title="删除">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

// Show DNS provider modal
function showDNSProviderModal() {
    ModalManager.show('dns-provider-modal', {
        title: '添加DNS供应商',
        defaults: {
            'dns-provider-id': ''
        }
    });
}

// Close DNS provider modal
function closeDNSProviderModal() {
    ModalManager.hide('dns-provider-modal');
}

// Edit DNS provider
async function editDNSProvider(id) {
    try {
        const provider = await API.post('/dns/provider/get', { id: id });

        ModalManager.show('dns-provider-modal', {
            title: '编辑DNS供应商',
            resetForm: false,
            defaults: {
                'dns-provider-id': provider.id,
                'dns-provider-name': provider.name,
                'dns-provider-server': provider.server,
                'dns-provider-type': provider.server_type,
                'dns-provider-default': provider.is_default
            }
        });
    } catch (error) {
        console.error('Failed to load DNS provider:', error);
        showToast('加载DNS供应商详情失败', 'error');
    }
}

// Delete DNS provider
async function deleteDNSProvider(id) {
    await deleteItem('/dns/provider/remove', id, 'DNS供应商', loadDNSProviders);
}

// Submit DNS provider form
async function submitDNSProvider(event) {
    event.preventDefault();

    const id = document.getElementById('dns-provider-id').value;

    const data = {
        name: document.getElementById('dns-provider-name').value,
        server: document.getElementById('dns-provider-server').value,
        server_type: document.getElementById('dns-provider-type').value,
        is_default: document.getElementById('dns-provider-default').checked
    };

    try {
        const endpoint = id ? '/dns/provider/update' : '/dns/provider/add';
        const body = id ? { ...data, id: parseInt(id) } : data;

        await API.post(endpoint, body);

        showToast(id ? 'DNS供应商已更新' : 'DNS供应商已添加', 'success');
        closeDNSProviderModal();
        loadDNSProviders();
    } catch (error) {
        console.error('Failed to submit DNS provider:', error);
        showToast('保存DNS供应商失败', 'error');
    }
}

// Initialize DNS providers when switching to settings tab
const originalSwitchTab = switchTab;
switchTab = function(tabName) {
    originalSwitchTab.call(this, tabName);

    if (tabName === 'settings' && dnsProviders.length === 0) {
        loadDNSProviders();
    }

    if (tabName === 'alerts') {
        loadAlertChannels();
        loadAlertRules();
    }
};

// Alert Channel Management

let alertChannels = [];
let alertRules = [];

// Load alert channels
async function loadAlertChannels() {
    try {
        const data = await API.post('/alert/channel/list');
        alertChannels = data.channels || [];
        renderAlertChannels();
    } catch (error) {
        console.error('Failed to load alert channels:', error);
        showToast('加载告警通道列表失败', 'error');
    }
}

// Render alert channels table
function renderAlertChannels() {
    const tbody = document.getElementById('alert-channels-tbody');

    if (alertChannels.length === 0) {
        renderEmptyState(tbody, 4, '暂无告警通道，点击上方"添加告警通道"按钮开始添加');
        return;
    }

    const typeLabels = {
        'wechat': '企业微信',
        'dingtalk': '钉钉',
        'telegram': 'Telegram',
        'email': '邮件'
    };

    tbody.innerHTML = alertChannels.map(channel => {
        const typeLabel = typeLabels[channel.type] || channel.type;
        const enabledBadge = channel.enabled
            ? '<span style="color: #10b981; font-weight: 600;"><i class="fas fa-check-circle"></i> 启用</span>'
            : '<span style="color: #6b7280; font-weight: 600;"><i class="fas fa-times-circle"></i> 禁用</span>';

        return `
            <tr data-id="${channel.id}">
                <td><strong>${channel.name}</strong></td>
                <td>${typeLabel}</td>
                <td>${enabledBadge}</td>
                <td>
                    <div style="display: flex; gap: 6px;">
                        <button class="btn btn-sm btn-secondary" onclick="testAlertChannel(${channel.id})" title="测试">
                            <i class="fas fa-paper-plane"></i>
                        </button>
                        <button class="btn btn-sm btn-secondary" onclick="editAlertChannel(${channel.id})" title="编辑">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="deleteAlertChannel(${channel.id})" title="删除">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

// Show alert channel modal
function showAlertChannelModal() {
    ModalManager.show('alert-channel-modal', {
        title: '添加告警通道',
        defaults: {
            'alert-channel-id': '',
            'alert-channel-enabled': true
        },
        onShow: () => {
            updateAlertChannelFields();
        }
    });
}

// Close alert channel modal
function closeAlertChannelModal() {
    ModalManager.hide('alert-channel-modal');
}

// Update alert channel fields based on type
function updateAlertChannelFields() {
    const type = document.getElementById('alert-channel-type').value;

    // Hide all sections
    document.getElementById('alert-wechat-section').style.display = 'none';
    document.getElementById('alert-dingtalk-section').style.display = 'none';
    document.getElementById('alert-telegram-section').style.display = 'none';
    document.getElementById('alert-email-section').style.display = 'none';

    // Show selected section
    const sectionId = `alert-${type}-section`;
    document.getElementById(sectionId).style.display = 'block';
}

// Edit alert channel
async function editAlertChannel(id) {
    try {
        const channel = await API.post('/alert/channel/get', { id: id });

        // Parse and set config
        const config = JSON.parse(channel.config || '{}');
        updateAlertChannelFields();

        const defaults = {
            'alert-channel-id': channel.id,
            'alert-channel-name': channel.name,
            'alert-channel-type': channel.type,
            'alert-channel-enabled': channel.enabled
        };

        if (channel.type === 'wechat') {
            defaults['wechat-webhook'] = config.webhook_url || '';
        } else if (channel.type === 'dingtalk') {
            defaults['dingtalk-webhook'] = config.webhook_url || '';
            defaults['dingtalk-secret'] = config.secret || '';
        } else if (channel.type === 'telegram') {
            defaults['telegram-bot-token'] = config.bot_token || '';
            defaults['telegram-chat-id'] = config.chat_id || '';
        } else if (channel.type === 'email') {
            defaults['email-smtp-host'] = config.smtp_host || '';
            defaults['email-smtp-port'] = config.smtp_port || 587;
            defaults['email-username'] = config.username || '';
            defaults['email-password'] = config.password || '';
            defaults['email-from'] = config.from || '';
            defaults['email-to'] = (config.to || []).join(', ');
            defaults['email-use-tls'] = config.use_tls !== false;
        }

        ModalManager.show('alert-channel-modal', {
            title: '编辑告警通道',
            resetForm: false,
            defaults: defaults
        });
    } catch (error) {
        console.error('Failed to load alert channel:', error);
        showToast('加载告警通道详情失败', 'error');
    }
}

// Delete alert channel
async function deleteAlertChannel(id) {
    await deleteItem('/alert/channel/remove', id, '告警通道', loadAlertChannels);
}

// Test alert channel
async function testAlertChannel(id) {
    try {
        await API.post('/alert/channel/test', { id: id });
        showToast('测试消息已发送，请检查是否收到', 'success');
    } catch (error) {
        console.error('Failed to test alert channel:', error);
        showToast('测试告警通道失败', 'error');
    }
}

// Submit alert channel form
async function submitAlertChannel(event) {
    event.preventDefault();

    const id = document.getElementById('alert-channel-id').value;
    const type = document.getElementById('alert-channel-type').value;

    const data = {
        name: document.getElementById('alert-channel-name').value,
        type: type,
        enabled: document.getElementById('alert-channel-enabled').checked
    };

    // Build config based on type
    const config = {};
    if (type === 'wechat') {
        config.webhook_url = document.getElementById('wechat-webhook').value;
    } else if (type === 'dingtalk') {
        config.webhook_url = document.getElementById('dingtalk-webhook').value;
        config.secret = document.getElementById('dingtalk-secret').value;
    } else if (type === 'telegram') {
        config.bot_token = document.getElementById('telegram-bot-token').value;
        config.chat_id = document.getElementById('telegram-chat-id').value;
    } else if (type === 'email') {
        config.smtp_host = document.getElementById('email-smtp-host').value;
        config.smtp_port = parseInt(document.getElementById('email-smtp-port').value);
        config.username = document.getElementById('email-username').value;
        config.password = document.getElementById('email-password').value;
        config.from = document.getElementById('email-from').value;
        config.to = document.getElementById('email-to').value.split(',').map(e => e.trim());
        config.use_tls = document.getElementById('email-use-tls').checked;
    }

    data.config = JSON.stringify(config);

    try {
        const endpoint = id ? '/alert/channel/update' : '/alert/channel/add';
        const body = id ? { ...data, id: parseInt(id) } : data;

        await API.post(endpoint, body);

        showToast(id ? '告警通道已更新' : '告警通道已添加', 'success');
        closeAlertChannelModal();
        loadAlertChannels();
    } catch (error) {
        console.error('Failed to submit alert channel:', error);
        showToast('保存告警通道失败', 'error');
    }
}

// Alert Rule Management

// Load alert rules
async function loadAlertRules() {
    try {
        const data = await API.post('/alert/rule/list');
        alertRules = data.rules || [];
        renderAlertRules();
    } catch (error) {
        console.error('Failed to load alert rules:', error);
        showToast('加载告警规则列表失败', 'error');
    }
}

// Render alert rules table
function renderAlertRules() {
    const tbody = document.getElementById('alert-rules-tbody');

    if (alertRules.length === 0) {
        renderEmptyState(tbody, 6, '暂无告警规则，点击上方"添加告警规则"按钮开始添加');
        return;
    }

    tbody.innerHTML = alertRules.map(rule => {
        const thresholdTypeLabels = {
            'failure_count': '故障次数',
            'response_time': '响应时间'
        };

        return `
            <tr data-id="${rule.id}">
                <td>${rule.target_name || '-'}</td>
                <td>${rule.channel_name || '-'}</td>
                <td>${thresholdTypeLabels[rule.threshold_type] || rule.threshold_type}</td>
                <td>${rule.threshold_value}</td>
                <td>${rule.enabled ? '<span style="color: #10b981;"><i class="fas fa-check-circle"></i></span>' : '<span style="color: #6b7280;"><i class="fas fa-times-circle"></i></span>'}</td>
                <td>
                    <div style="display: flex; gap: 6px;">
                        <button class="btn btn-sm btn-secondary" onclick="editAlertRule(${rule.id})" title="编辑">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="deleteAlertRule(${rule.id})" title="删除">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

// Show alert rule modal
async function showAlertRuleModal() {
    // Load monitors and channels
    await loadMonitorOptions();
    await loadChannelOptions();

    ModalManager.show('alert-rule-modal', {
        title: '添加告警规则',
        defaults: {
            'alert-rule-id': '',
            'alert-rule-enabled': true
        }
    });
}

// Load monitor options for alert rule
async function loadMonitorOptions() {
    try {
        const data = await API.post('/monitor/list');
        const monitors = data.targets || [];
        populateSelect('alert-rule-target', monitors, '请选择监控目标', 'id', 'name');
    } catch (error) {
        console.error('Failed to load monitors:', error);
    }
}

// Load channel options for alert rule
async function loadChannelOptions() {
    try {
        const data = await API.post('/alert/channel/list');
        const channels = data.channels || [];
        populateSelect('alert-rule-channel', channels, '请选择告警通道', 'id', 'name');
    } catch (error) {
        console.error('Failed to load channels:', error);
    }
}

// Close alert rule modal
function closeAlertRuleModal() {
    ModalManager.hide('alert-rule-modal');
}

// Edit alert rule
async function editAlertRule(id) {
    try {
        const rule = await API.post('/alert/rule/get', { id: id });

        // Load monitors and channels
        await loadMonitorOptions();
        await loadChannelOptions();

        ModalManager.show('alert-rule-modal', {
            title: '编辑告警规则',
            resetForm: false,
            defaults: {
                'alert-rule-id': rule.id,
                'alert-rule-target': rule.target_id,
                'alert-rule-channel': rule.channel_id,
                'alert-rule-threshold-type': rule.threshold_type,
                'alert-rule-threshold-value': rule.threshold_value,
                'alert-rule-enabled': rule.enabled
            }
        });
    } catch (error) {
        console.error('Failed to load alert rule:', error);
        showToast('加载告警规则详情失败', 'error');
    }
}

// Delete alert rule
async function deleteAlertRule(id) {
    await deleteItem('/alert/rule/remove', id, '告警规则', loadAlertRules);
}

// Submit alert rule form
async function submitAlertRule(event) {
    event.preventDefault();

    const id = document.getElementById('alert-rule-id').value;

    const data = {
        target_id: parseInt(document.getElementById('alert-rule-target').value),
        channel_id: parseInt(document.getElementById('alert-rule-channel').value),
        threshold_type: document.getElementById('alert-rule-threshold-type').value,
        threshold_value: parseInt(document.getElementById('alert-rule-threshold-value').value),
        enabled: document.getElementById('alert-rule-enabled').checked
    };

    try {
        const endpoint = id ? '/alert/rule/update' : '/alert/rule/add';
        const body = id ? { ...data, id: parseInt(id) } : data;

        await API.post(endpoint, body);

        showToast(id ? '告警规则已更新' : '告警规则已添加', 'success');
        closeAlertRuleModal();
        loadAlertRules();
    } catch (error) {
        console.error('Failed to submit alert rule:', error);
        showToast('保存告警规则失败', 'error');
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
    const resolvedHostInput = document.getElementById('monitor-resolved-host');
    const dnsServerNameInput = document.getElementById('monitor-dns-server-name');
    const dnsServerInput = document.getElementById('monitor-dns-server');
    const dnsServerTypeInput = document.getElementById('monitor-dns-server-type');

    if (!providerId) {
        // User selected "手动输入", enable custom host field
        resolvedHostInput.disabled = false;
        resolvedHostInput.placeholder = "例如: example.com 或 192.168.1.1";
        return;
    }

    // User selected a DNS provider, disable custom host field
    resolvedHostInput.disabled = true;
    resolvedHostInput.value = '';
    resolvedHostInput.placeholder = "已选择DNS供应商，无需手动配置";

    // Find the selected provider from the loaded providers
    const provider = dnsProviders.find(p => p.id == providerId);
    if (provider) {
        dnsServerNameInput.value = provider.name;
        dnsServerInput.value = provider.server;
        dnsServerTypeInput.value = provider.server_type;
    }
}

// Load alert channels for monitor dropdown
async function loadAlertChannelsForMonitor() {
    try {
        const data = await API.post('/alert/channel/list');
        const channels = data.channels || [];

        const select = document.getElementById('monitor-alert-channels');
        select.innerHTML = '';

        if (channels.length === 0) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = '暂无告警通道';
            option.disabled = true;
            select.appendChild(option);
        } else {
            channels.forEach(channel => {
                const option = document.createElement('option');
                option.value = channel.id;
                option.textContent = channel.name;
                select.appendChild(option);
            });
        }
    } catch (error) {
        console.error('Failed to load alert channels for monitor:', error);
    }
}
// System Configuration Management

let systemConfig = null;

// Load system configuration
async function loadSystemConfig() {
    console.log('Loading system config from /api/v1/config...');
    try {
        const data = await API.get('/config');
        console.log('Config data received:', data);
        // 后端返回的是 {config: {...}} 格式
        systemConfig = data.Config || data.config;
        if (!systemConfig) {
            throw new Error('Config not found in response');
        }
        displaySystemConfig(systemConfig);
        showToast('配置加载成功', 'success');
    } catch (error) {
        console.error('Failed to load system config:', error);
        showToast('加载系统配置失败: ' + (error.message || '未知错误'), 'error');
        // 尝试显示错误详情
        if (error.message) {
            console.error('Error message:', error.message);
        }
        if (error.stack) {
            console.error('Error stack:', error.stack);
        }
    }
}

// Display system configuration
function displaySystemConfig(config) {
    const container = document.getElementById('config-display');
    if (!container) {
        console.error('config-display container not found');
        return;
    }

    const html = `
        <div class="config-grid">
            <div class="config-group">
                <h4>服务器配置</h4>
                <p><strong>HTTP 端口:</strong> ${config.Server.HTTPPort}</p>
                <p><strong>gRPC 端口:</strong> ${config.Server.GRPCPort}</p>
                <p><strong>监听地址:</strong> ${config.Server.Host}</p>
            </div>
            <div class="config-group">
                <h4>数据库配置</h4>
                <p><strong>类型:</strong> ${config.Database.Driver}</p>
                <p><strong>主机:</strong> ${config.Database.Host || '-'}</p>
                <p><strong>端口:</strong> ${config.Database.Port || '-'}</p>
                <p><strong>数据库:</strong> ${config.Database.DBName}</p>
                <p><strong>用户:</strong> ${config.Database.User || '-'}</p>
            </div>
            <div class="config-group">
                <h4>监控配置</h4>
                <p><strong>检查间隔:</strong> ${config.Monitor.CheckInterval} 秒</p>
                <p><strong>工作线程:</strong> ${config.Monitor.Workers}</p>
            </div>
            <div class="config-group">
                <h4>日志配置</h4>
                <p><strong>级别:</strong> ${config.Logger.Level}</p>
                <p><strong>输出:</strong> ${config.Logger.Output}</p>
            </div>
            <div class="config-group">
                <h4>Elasticsearch 配置</h4>
                <p><strong>状态:</strong> ${config.Elasticsearch.Enabled ? '启用' : '禁用'}</p>
                <p><strong>地址:</strong> ${config.Elasticsearch.Addresses.join(', ')}</p>
                <p><strong>索引前缀:</strong> ${config.Elasticsearch.IndexPrefix}</p>
            </div>
            <div class="config-group">
                <h4>告警配置</h4>
                <p><strong>状态:</strong> ${config.Alert.Enabled ? '启用' : '禁用'}</p>
                <p><strong>冷却时间:</strong> ${config.Alert.CooldownSeconds} 秒</p>
                <p><strong>重试次数:</strong> ${config.Alert.RetryTimes}</p>
            </div>
        </div>
    `;

    container.innerHTML = html;
    console.log('Config displayed successfully');
}

// Show system config modal
function showSystemConfigModal() {
    console.log('Opening system config modal...');
    if (!systemConfig) {
        console.log('Loading system config first...');
        loadSystemConfig().then(() => {
            if (systemConfig) {
                populateSystemConfigForm();
                document.body.classList.add('modal-open');
                console.log('Showing system config modal');
                ModalManager.show('system-config-modal', { resetForm: false });
            } else {
                showToast('无法加载系统配置', 'error');
            }
        }).catch(err => {
            console.error('Error loading config:', err);
            showToast('加载系统配置失败', 'error');
        });
    } else {
        populateSystemConfigForm();
        document.body.classList.add('modal-open');
        console.log('Showing system config modal');
        ModalManager.show('system-config-modal', { resetForm: false });
    }
}

// Close system config modal
function closeSystemConfigModal() {
    document.body.classList.remove('modal-open');
    ModalManager.hide('system-config-modal');
}

// Populate system config form
function populateSystemConfigForm() {
    if (!systemConfig) {
        console.error('No system config available');
        return;
    }

    console.log('Populating form with config:', systemConfig);

    try {
        // Server config
        document.getElementById('config-http-port').value = systemConfig.Server.HTTPPort;
        document.getElementById('config-grpc-port').value = systemConfig.Server.GRPCPort;
        document.getElementById('config-host').value = systemConfig.Server.Host;

        // Database config
        document.getElementById('config-db-driver').value = systemConfig.Database.Driver;
        document.getElementById('config-db-host').value = systemConfig.Database.Host || '';
        document.getElementById('config-db-port').value = systemConfig.Database.Port || 3306;
        document.getElementById('config-db-user').value = systemConfig.Database.User || '';
        document.getElementById('config-db-password').value = ''; // Don't show password
        document.getElementById('config-db-name').value = systemConfig.Database.DBName;
        document.getElementById('config-db-sslmode').value = systemConfig.Database.SSLMode || 'disable';

        // Elasticsearch config
        document.getElementById('config-es-enabled').checked = systemConfig.Elasticsearch.Enabled;
        document.getElementById('config-es-addresses').value = systemConfig.Elasticsearch.Addresses.join(',');
        document.getElementById('config-es-username').value = systemConfig.Elasticsearch.Username || '';
        document.getElementById('config-es-password').value = ''; // Don't show existing password
        document.getElementById('config-es-prefix').value = systemConfig.Elasticsearch.IndexPrefix;

        // Monitor config
        document.getElementById('config-monitor-interval').value = systemConfig.Monitor.CheckInterval;
        document.getElementById('config-monitor-workers').value = systemConfig.Monitor.Workers;

        // Logger config
        document.getElementById('config-log-level').value = systemConfig.Logger.Level;
        document.getElementById('config-log-output').value = systemConfig.Logger.Output;

        updateDatabaseFields();
        console.log('Form populated successfully');
    } catch (error) {
        console.error('Error populating form:', error);
        showToast('填充表单失败: ' + error.message, 'error');
    }
}

// Update database fields based on driver
function updateDatabaseFields() {
    const driver = document.getElementById('config-db-driver').value;
    const isSqlite = driver === 'sqlite';

    document.getElementById('db-host-group').style.display = isSqlite ? 'none' : 'block';
    document.getElementById('db-port-group').style.display = isSqlite ? 'none' : 'block';
    document.getElementById('db-user-group').style.display = isSqlite ? 'none' : 'block';
    document.getElementById('db-password-group').style.display = isSqlite ? 'none' : 'block';
    document.getElementById('db-sslmode-group').style.display = isSqlite ? 'none' : 'block';
}

// Test database connection
async function testDatabaseConnection() {
    const req = {
        driver: document.getElementById('config-db-driver').value,
        host: document.getElementById('config-db-host').value,
        port: parseInt(document.getElementById('config-db-port').value) || 3306,
        user: document.getElementById('config-db-user').value,
        password: document.getElementById('config-db-password').value,
        dbname: document.getElementById('config-db-name').value,
        sslmode: document.getElementById('config-db-sslmode').value
    };

    try {
        const data = await API.post('/config/testDatabase', req);

        if (data.message) {
            showToast(data.message, 'success');
        } else {
            showToast(data.error || '测试失败', 'error');
        }
    } catch (error) {
        console.error('Failed to test database:', error);
        showToast('测试数据库连接失败', 'error');
    }
}

// Submit system config
async function submitSystemConfig(event) {
    event.preventDefault();

    const config = {
        Server: {
            HTTPPort: parseInt(document.getElementById('config-http-port').value),
            GRPCPort: parseInt(document.getElementById('config-grpc-port').value),
            Host: document.getElementById('config-host').value
        },
        Database: {
            Driver: document.getElementById('config-db-driver').value,
            Host: document.getElementById('config-db-host').value,
            Port: parseInt(document.getElementById('config-db-port').value) || 0,
            User: document.getElementById('config-db-user').value,
            Password: document.getElementById('config-db-password').value,
            DBName: document.getElementById('config-db-name').value,
            SSLMode: document.getElementById('config-db-sslmode').value
        },
        Elasticsearch: {
            Enabled: document.getElementById('config-es-enabled').checked,
            Addresses: document.getElementById('config-es-addresses').value.split(',').map(a => a.trim()),
            Username: document.getElementById('config-es-username').value,
            Password: document.getElementById('config-es-password').value,
            IndexPrefix: document.getElementById('config-es-prefix').value
        },
        Monitor: {
            CheckInterval: parseInt(document.getElementById('config-monitor-interval').value),
            Workers: parseInt(document.getElementById('config-monitor-workers').value)
        },
        Logger: {
            Level: document.getElementById('config-log-level').value,
            Output: document.getElementById('config-log-output').value
        },
        Alert: systemConfig ? systemConfig.Alert : {
            Enabled: true,
            CooldownSeconds: 300,
            RetryTimes: 3,
            RetryInterval: 60
        },
        SNMP: systemConfig ? systemConfig.SNMP : {
            DefaultCommunity: 'public',
            DefaultVersion: 'v2c',
            DefaultTimeout: 5000
        }
    };

    try {
        // 后端期望 {config: {...}} 格式
        const data = await API.post('/config', { config: config });

        showToast(data.message, 'success');
        closeSystemConfigModal();
        loadSystemConfig();

        // 自动重启服务
        try {
            const restartData = await API.post('/config/restart', {});
            if (restartData.message) {
                showToast(restartData.message, 'info');
                setTimeout(() => {
                    location.reload();
                }, 3000);
            }
        } catch (restartError) {
            console.error('Failed to restart service:', restartError);
            showToast('配置已保存，但自动重启失败，请手动重启服务', 'warning');
        }
    } catch (error) {
        console.error('Failed to save config:', error);
        showToast('保存配置失败', 'error');
    }
}
