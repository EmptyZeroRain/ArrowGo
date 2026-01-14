// Settings Page - System Configuration and DNS Provider Management

let systemConfig = null;
let dnsProviders = [];

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadDNSProviders();
});

// DNS Provider Management

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

// System Configuration Management

// Load system configuration
async function loadSystemConfig() {
    try {
        const data = await API.get('/config');
        systemConfig = data.config;
        displaySystemConfig(systemConfig);
        showToast('配置加载成功', 'success');
    } catch (error) {
        console.error('Failed to load system config:', error);
        showToast('加载系统配置失败', 'error');
    }
}

// Display system configuration
function displaySystemConfig(config) {
    const container = document.getElementById('config-display');

    const html = `
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px;">
            <div>
                <h4 style="margin-bottom: 10px;">服务器配置</h4>
                <p><strong>HTTP 端口:</strong> ${config.server.http_port}</p>
                <p><strong>gRPC 端口:</strong> ${config.server.grpc_port}</p>
                <p><strong>监听地址:</strong> ${config.server.host}</p>
            </div>
            <div>
                <h4 style="margin-bottom: 10px;">数据库配置</h4>
                <p><strong>类型:</strong> ${config.database.driver}</p>
                <p><strong>主机:</strong> ${config.database.host || '-'}</p>
                <p><strong>端口:</strong> ${config.database.port || '-'}</p>
                <p><strong>数据库:</strong> ${config.database.dbname}</p>
                <p><strong>用户:</strong> ${config.database.user || '-'}</p>
            </div>
            <div>
                <h4 style="margin-bottom: 10px;">监控配置</h4>
                <p><strong>检查间隔:</strong> ${config.monitor.check_interval} 秒</p>
                <p><strong>工作线程:</strong> ${config.monitor.workers}</p>
            </div>
            <div>
                <h4 style="margin-bottom: 10px;">日志配置</h4>
                <p><strong>级别:</strong> ${config.logger.level}</p>
                <p><strong>输出:</strong> ${config.logger.output}</p>
            </div>
            <div>
                <h4 style="margin-bottom: 10px;">Elasticsearch 配置</h4>
                <p><strong>状态:</strong> ${config.elasticsearch.enabled ? '启用' : '禁用'}</p>
                <p><strong>地址:</strong> ${config.elasticsearch.addresses.join(', ')}</p>
                <p><strong>索引前缀:</strong> ${config.elasticsearch.index_prefix}</p>
            </div>
            <div>
                <h4 style="margin-bottom: 10px;">告警配置</h4>
                <p><strong>状态:</strong> ${config.alert.enabled ? '启用' : '禁用'}</p>
                <p><strong>冷却时间:</strong> ${config.alert.cooldown_seconds} 秒</p>
                <p><strong>重试次数:</strong> ${config.alert.retry_times}</p>
            </div>
        </div>
    `;

    container.innerHTML = html;
}

// Show system config modal
function showSystemConfigModal() {
    if (!systemConfig) {
        loadSystemConfig().then(() => {
            if (systemConfig) {
                populateSystemConfigForm();
                ModalManager.show('system-config-modal', { resetForm: false });
            }
        });
    } else {
        populateSystemConfigForm();
        ModalManager.show('system-config-modal', { resetForm: false });
    }
}

// Close system config modal
function closeSystemConfigModal() {
    ModalManager.hide('system-config-modal');
}

// Populate system config form
function populateSystemConfigForm() {
    if (!systemConfig) return;

    // Server config
    document.getElementById('config-host').value = systemConfig.server.host;
    document.getElementById('config-http-port').value = systemConfig.server.http_port;
    document.getElementById('config-grpc-port').value = systemConfig.server.grpc_port;

    // Database config
    document.getElementById('config-db-driver').value = systemConfig.database.driver;
    document.getElementById('config-db-host').value = systemConfig.database.host || '';
    document.getElementById('config-db-port').value = systemConfig.database.port || 3306;
    document.getElementById('config-db-user').value = systemConfig.database.user || '';
    document.getElementById('config-db-name').value = systemConfig.database.dbname;
    document.getElementById('config-db-sslmode').value = systemConfig.database.sslmode || 'disable';

    // Elasticsearch config
    document.getElementById('config-es-enabled').checked = systemConfig.elasticsearch.enabled;
    document.getElementById('config-es-addresses').value = systemConfig.elasticsearch.addresses.join(',');
    document.getElementById('config-es-username').value = systemConfig.elasticsearch.username || '';
    document.getElementById('config-es-password').value = '';
    document.getElementById('config-es-prefix').value = systemConfig.elasticsearch.index_prefix;

    // Monitor config
    document.getElementById('config-monitor-interval').value = systemConfig.monitor.check_interval;
    document.getElementById('config-monitor-workers').value = systemConfig.monitor.workers;

    // Logger config
    document.getElementById('config-log-level').value = systemConfig.logger.level;
    document.getElementById('config-log-output').value = systemConfig.logger.output;

    updateDatabaseFields();
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
        server: {
            http_port: parseInt(document.getElementById('config-http-port').value),
            grpc_port: parseInt(document.getElementById('config-grpc-port').value),
            host: document.getElementById('config-host').value
        },
        database: {
            driver: document.getElementById('config-db-driver').value,
            host: document.getElementById('config-db-host').value,
            port: parseInt(document.getElementById('config-db-port').value) || 0,
            user: document.getElementById('config-db-user').value,
            password: document.getElementById('config-db-password').value,
            dbname: document.getElementById('config-db-name').value,
            sslmode: document.getElementById('config-db-sslmode').value
        },
        elasticsearch: {
            enabled: document.getElementById('config-es-enabled').checked,
            addresses: document.getElementById('config-es-addresses').value.split(',').map(a => a.trim()),
            username: document.getElementById('config-es-username').value,
            password: document.getElementById('config-es-password').value,
            index_prefix: document.getElementById('config-es-prefix').value
        },
        monitor: {
            check_interval: parseInt(document.getElementById('config-monitor-interval').value),
            workers: parseInt(document.getElementById('config-monitor-workers').value)
        },
        logger: {
            level: document.getElementById('config-log-level').value,
            output: document.getElementById('config-log-output').value
        },
        alert: systemConfig ? systemConfig.alert : {
            enabled: true,
            cooldown_seconds: 300,
            retry_times: 3,
            retry_interval: 60
        },
        snmp: systemConfig ? systemConfig.snmp : {
            default_community: 'public',
            default_version: 'v2c',
            default_timeout: 5000
        }
    };

    try {
        const data = await API.post('/config', { config: config });

        showToast(data.message, 'success');
        closeSystemConfigModal();
        loadSystemConfig();

        // Auto restart service
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

// Query IP geolocation (moved to settings page)
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

// Setup modal backdrop clicks
ModalManager.setupBackdropClick('dns-provider-modal', closeDNSProviderModal);
ModalManager.setupBackdropClick('system-config-modal', closeSystemConfigModal);

// Close modals on escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeDNSProviderModal();
        closeSystemConfigModal();
    }
});
