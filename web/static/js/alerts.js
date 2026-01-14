// Alerts Page - Alert Channel and Rule Management

let alertChannels = [];
let alertRules = [];

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    loadAlertChannels();
    loadAlertRules();
});

// Alert Channel Management

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

    const thresholdTypeLabels = {
        'failure_count': '故障次数',
        'response_time': '响应时间'
    };

    tbody.innerHTML = alertRules.map(rule => {
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

// Setup modal backdrop clicks
ModalManager.setupBackdropClick('alert-channel-modal', closeAlertChannelModal);
ModalManager.setupBackdropClick('alert-rule-modal', closeAlertRuleModal);

// Close modals on escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeAlertChannelModal();
        closeAlertRuleModal();
    }
});
