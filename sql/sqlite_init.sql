-- ============================================
-- Monitor 监控拨测系统 - SQLite 初始化脚本
-- 版本: v3.0
-- 更新日期: 2025-01-14
-- ============================================

-- SQLite 专用 PRAGMA 设置
PRAGMA foreign_keys = ON;
PRAGMA encoding = 'UTF-8';

-- ============================================
-- 1. 监控目标表 (monitor_targets)
-- ============================================
CREATE TABLE IF NOT EXISTS monitor_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,           -- http, https, tcp, udp, dns
    address VARCHAR(500) NOT NULL,
    port INTEGER,
    interval INTEGER DEFAULT 60,         -- 检查间隔（秒）
    metadata TEXT,                       -- JSON 字符串
    enabled BOOLEAN DEFAULT 1,

    -- HTTP/HTTPS 专用字段
    http_method VARCHAR(10),             -- GET, POST, PUT, DELETE
    http_headers TEXT,                   -- JSON 字符串
    http_body TEXT,
    resolved_host VARCHAR(255),          -- 自定义 host 解析
    follow_redirects BOOLEAN DEFAULT 1,
    max_redirects INTEGER DEFAULT 10,
    expected_status_codes TEXT,          -- 逗号分隔的状态码

    -- DNS 专用字段
    dns_server VARCHAR(255),
    dns_server_name VARCHAR(255),
    dns_server_type VARCHAR(10),         -- udp, tcp, doh, dot

    -- PING 专用字段
    ping_count INTEGER DEFAULT 4,
    ping_size INTEGER DEFAULT 32,
    ping_timeout INTEGER DEFAULT 5000,

    -- SMTP 专用字段
    smtp_username VARCHAR(255),
    smtp_password VARCHAR(255),
    smtp_use_tls BOOLEAN DEFAULT 0,
    smtp_mail_from VARCHAR(255),
    smtp_mail_to VARCHAR(255),
    smtp_check_starttls BOOLEAN DEFAULT 1,

    -- SNMP 专用字段
    snmp_community VARCHAR(255),
    snmp_oid VARCHAR(500),
    snmp_version VARCHAR(10),
    snmp_expected_value VARCHAR(255),
    snmp_operator VARCHAR(10),

    -- SSL/TLS 证书专用字段
    ssl_warn_days INTEGER DEFAULT 30,
    ssl_critical_days INTEGER DEFAULT 7,
    ssl_get_chain BOOLEAN DEFAULT 1,
    ssl_check BOOLEAN DEFAULT 0,

    -- 告警渠道关联
    alert_channel_ids TEXT,              -- JSON 数组

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_monitor_targets_type ON monitor_targets(type);
CREATE INDEX IF NOT EXISTS idx_monitor_targets_enabled ON monitor_targets(enabled);

-- ============================================
-- 2. 监控状态表 (monitor_status)
-- ============================================
CREATE TABLE IF NOT EXISTS monitor_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,         -- up, down, unknown
    response_time INTEGER,               -- 毫秒
    message TEXT,
    checked_at DATETIME,
    uptime_percentage INTEGER DEFAULT 0,

    -- SSL 证书信息
    ssl_days_until_expiry INTEGER,
    ssl_issuer VARCHAR(255),
    ssl_subject VARCHAR(255),
    ssl_serial VARCHAR(128),

    -- DNS 信息
    dns_records TEXT,                    -- JSON 字符串
    resolved_ip VARCHAR(64),

    -- 额外的检查数据
    data TEXT,                           -- 完整的检查结果数据

    FOREIGN KEY (target_id) REFERENCES monitor_targets(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_monitor_status_target_id ON monitor_status(target_id);
CREATE INDEX IF NOT EXISTS idx_monitor_status_checked_at ON monitor_status(checked_at);
CREATE INDEX IF NOT EXISTS idx_monitor_status_status ON monitor_status(status);

-- ============================================
-- 3. 监控历史表 (monitor_history)
-- ============================================
CREATE TABLE IF NOT EXISTS monitor_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    response_time INTEGER,
    message TEXT,
    checked_at DATETIME
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_monitor_history_target_id ON monitor_history(target_id);
CREATE INDEX IF NOT EXISTS idx_monitor_history_checked_at ON monitor_history(checked_at);

-- ============================================
-- 4. IP 地理位置缓存表 (ip_geo_cache)
-- ============================================
CREATE TABLE IF NOT EXISTS ip_geo_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip VARCHAR(45) NOT NULL UNIQUE,      -- IPv4 或 IPv6
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    isp VARCHAR(255),
    latitude REAL,
    longitude REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ip_geo_cache_ip ON ip_geo_cache(ip);

-- ============================================
-- 5. DNS 供应商表 (dns_providers)
-- ============================================
CREATE TABLE IF NOT EXISTS dns_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    server VARCHAR(500) NOT NULL,        -- DNS 服务器地址
    server_type VARCHAR(10) NOT NULL,    -- udp, tcp, doh, dot
    is_default BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 6. 告警渠道表 (alert_channels)
-- ============================================
CREATE TABLE IF NOT EXISTS alert_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,           -- email, webhook, dingtalk, wechat
    enabled BOOLEAN DEFAULT 1,
    config TEXT NOT NULL,                -- JSON 配置
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- 7. 告警规则表 (alert_rules)
-- ============================================
CREATE TABLE IF NOT EXISTS alert_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    target_id INTEGER NOT NULL,
    channel_id INTEGER NOT NULL,
    threshold_type VARCHAR(20),          -- failure_count, response_time
    threshold_value INTEGER,
    enabled BOOLEAN DEFAULT 1,

    -- 高级字段
    condition_logic TEXT,                -- JSON: 复杂条件
    cooldown_seconds INTEGER DEFAULT 300,
    last_alert_time DATETIME,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (target_id) REFERENCES monitor_targets(id) ON DELETE CASCADE,
    FOREIGN KEY (channel_id) REFERENCES alert_channels(id) ON DELETE CASCADE
);

-- ============================================
-- 8. 告警条件表 (alert_conditions)
-- ============================================
CREATE TABLE IF NOT EXISTS alert_conditions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL,
    field_type VARCHAR(50) NOT NULL,     -- status, response_time, uptime
    operator VARCHAR(10) NOT NULL,       -- eq, ne, gt, lt, ge, le, contains
    value TEXT,
    logical_op VARCHAR(5),               -- and, or
    order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
);

-- ============================================
-- 9. 告警规则组表 (alert_rule_groups)
-- ============================================
CREATE TABLE IF NOT EXISTS alert_rule_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL,
    name VARCHAR(255),
    logical_op VARCHAR(5),               -- and, or
    order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
);

-- ============================================
-- 10. 告警历史表 (alert_history)
-- ============================================
CREATE TABLE IF NOT EXISTS alert_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER,
    target_id INTEGER,
    channel_id INTEGER,
    severity VARCHAR(50),
    status VARCHAR(50),
    message TEXT,
    sent_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_alert_history_rule_id ON alert_history(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_target_id ON alert_history(target_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_sent_at ON alert_history(sent_at);

-- ============================================
-- 初始化数据
-- ============================================

-- 插入默认 DNS 供应商
INSERT OR IGNORE INTO dns_providers (name, server, server_type, is_default) VALUES
    ('Google DNS', '8.8.8.8:53', 'udp', 1),
    ('Cloudflare DNS', '1.1.1.1:53', 'udp', 0),
    ('阿里 DNS', '223.5.5.5:53', 'udp', 0),
    ('腾讯 DNS', '119.29.29.29:53', 'udp', 0),
    ('114 DNS', '114.114.114.114:53', 'udp', 0);

-- ============================================
-- 触发器（用于自动更新 updated_at 字段）
-- ============================================

CREATE TRIGGER IF NOT EXISTS update_monitor_targets_timestamp
AFTER UPDATE ON monitor_targets
FOR EACH ROW
BEGIN
    UPDATE monitor_targets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_ip_geo_cache_timestamp
AFTER UPDATE ON ip_geo_cache
FOR EACH ROW
BEGIN
    UPDATE ip_geo_cache SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_dns_providers_timestamp
AFTER UPDATE ON dns_providers
FOR EACH ROW
BEGIN
    UPDATE dns_providers SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_alert_channels_timestamp
AFTER UPDATE ON alert_channels
FOR EACH ROW
BEGIN
    UPDATE alert_channels SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_alert_rules_timestamp
AFTER UPDATE ON alert_rules
FOR EACH ROW
BEGIN
    UPDATE alert_rules SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================
-- 初始化完成
-- ============================================
-- 数据库初始化成功完成！
-- 表总数: 10
-- 索引: 已创建
-- 触发器: 已创建
-- 默认数据: 已插入
