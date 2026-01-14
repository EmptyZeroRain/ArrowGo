-- ============================================
-- Monitor 监控拨测系统 - PostgreSQL 初始化脚本
-- 版本: v3.0
-- 更新日期: 2025-01-14
-- 兼容: PostgreSQL 12+, PostgreSQL 14+
-- ============================================

-- 设置客户端编码
SET client_encoding = 'UTF8';

-- ============================================
-- 1. 监控目标表 (monitor_targets)
-- ============================================
DROP TABLE IF EXISTS monitor_targets CASCADE;
CREATE TABLE monitor_targets (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,           -- http, https, tcp, udp, dns
    address VARCHAR(500) NOT NULL,
    port INTEGER,
    interval BIGINT DEFAULT 60,          -- 检查间隔（秒）
    metadata TEXT,                       -- JSON 字符串
    enabled BOOLEAN DEFAULT true,

    -- HTTP/HTTPS 专用字段
    http_method VARCHAR(10),             -- GET, POST, PUT, DELETE
    http_headers TEXT,                   -- JSON 字符串
    http_body TEXT,
    resolved_host VARCHAR(255),          -- 自定义 host 解析
    follow_redirects BOOLEAN DEFAULT true,
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
    smtp_use_tls BOOLEAN DEFAULT false,
    smtp_mail_from VARCHAR(255),
    smtp_mail_to VARCHAR(255),
    smtp_check_starttls BOOLEAN DEFAULT true,

    -- SNMP 专用字段
    snmp_community VARCHAR(255),
    snmp_oid VARCHAR(500),
    snmp_version VARCHAR(10),
    snmp_expected_value VARCHAR(255),
    snmp_operator VARCHAR(10),

    -- SSL/TLS 证书专用字段
    ssl_warn_days INTEGER DEFAULT 30,
    ssl_critical_days INTEGER DEFAULT 7,
    ssl_get_chain BOOLEAN DEFAULT true,
    ssl_check BOOLEAN DEFAULT false,

    -- 告警渠道关联
    alert_channel_ids TEXT,              -- JSON 数组

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_monitor_targets_type ON monitor_targets(type);
CREATE INDEX idx_monitor_targets_enabled ON monitor_targets(enabled);

-- 添加注释
COMMENT ON TABLE monitor_targets IS '监控目标表';
COMMENT ON COLUMN monitor_targets.type IS '类型: http, https, tcp, udp, dns';
COMMENT ON COLUMN monitor_targets.interval IS '检查间隔（秒）';
COMMENT ON COLUMN monitor_targets.enabled IS '是否启用';

-- ============================================
-- 2. 监控状态表 (monitor_status)
-- ============================================
DROP TABLE IF EXISTS monitor_status CASCADE;
CREATE TABLE monitor_status (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,         -- up, down, unknown
    response_time BIGINT,                -- 毫秒
    message TEXT,
    checked_at TIMESTAMP WITH TIME ZONE,
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
CREATE INDEX idx_monitor_status_target_id ON monitor_status(target_id);
CREATE INDEX idx_monitor_status_checked_at ON monitor_status(checked_at);
CREATE INDEX idx_monitor_status_status ON monitor_status(status);

-- 添加注释
COMMENT ON TABLE monitor_status IS '监控状态表';
COMMENT ON COLUMN monitor_status.status IS '状态: up, down, unknown';
COMMENT ON COLUMN monitor_status.response_time IS '响应时间（毫秒）';

-- ============================================
-- 3. 监控历史表 (monitor_history)
-- ============================================
DROP TABLE IF EXISTS monitor_history CASCADE;
CREATE TABLE monitor_history (
    id BIGSERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    response_time BIGINT,
    message TEXT,
    checked_at TIMESTAMP WITH TIME ZONE,

    FOREIGN KEY (target_id) REFERENCES monitor_targets(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_monitor_history_target_id ON monitor_history(target_id);
CREATE INDEX idx_monitor_history_checked_at ON monitor_history(checked_at);

-- 添加注释
COMMENT ON TABLE monitor_history IS '监控历史表';

-- ============================================
-- 4. IP 地理位置缓存表 (ip_geo_cache)
-- ============================================
DROP TABLE IF EXISTS ip_geo_cache CASCADE;
CREATE TABLE ip_geo_cache (
    id BIGSERIAL PRIMARY KEY,
    ip VARCHAR(45) NOT NULL UNIQUE,      -- IPv4 或 IPv6
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    isp VARCHAR(255),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_ip_geo_cache_ip ON ip_geo_cache(ip);

-- 添加注释
COMMENT ON TABLE ip_geo_cache IS 'IP地理位置缓存表';
COMMENT ON COLUMN ip_geo_cache.ip IS 'IP地址（支持IPv4和IPv6）';

-- ============================================
-- 5. DNS 供应商表 (dns_providers)
-- ============================================
DROP TABLE IF EXISTS dns_providers CASCADE;
CREATE TABLE dns_providers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    server VARCHAR(500) NOT NULL,        -- DNS 服务器地址
    server_type VARCHAR(10) NOT NULL,    -- udp, tcp, doh, dot
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 添加注释
COMMENT ON TABLE dns_providers IS 'DNS供应商表';
COMMENT ON COLUMN dns_providers.server_type IS 'DNS协议类型: udp, tcp, doh, dot';

-- ============================================
-- 6. 告警渠道表 (alert_channels)
-- ============================================
DROP TABLE IF EXISTS alert_channels CASCADE;
CREATE TABLE alert_channels (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,           -- email, webhook, dingtalk, wechat
    enabled BOOLEAN DEFAULT true,
    config TEXT NOT NULL,                -- JSON 配置
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 添加注释
COMMENT ON TABLE alert_channels IS '告警渠道表';
COMMENT ON COLUMN alert_channels.type IS '渠道类型: email, webhook, dingtalk, wechat';

-- ============================================
-- 7. 告警规则表 (alert_rules)
-- ============================================
DROP TABLE IF EXISTS alert_rules CASCADE;
CREATE TABLE alert_rules (
    id BIGSERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL,
    channel_id INTEGER NOT NULL,
    threshold_type VARCHAR(20),          -- failure_count, response_time
    threshold_value INTEGER,
    enabled BOOLEAN DEFAULT true,

    -- 高级字段
    condition_logic TEXT,                -- JSON: 复杂条件
    cooldown_seconds INTEGER DEFAULT 300,
    last_alert_time TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (target_id) REFERENCES monitor_targets(id) ON DELETE CASCADE,
    FOREIGN KEY (channel_id) REFERENCES alert_channels(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_alert_rules_target_id ON alert_rules(target_id);
CREATE INDEX idx_alert_rules_channel_id ON alert_rules(channel_id);

-- 添加注释
COMMENT ON TABLE alert_rules IS '告警规则表';
COMMENT ON COLUMN alert_rules.threshold_type IS '阈值类型: failure_count, response_time';

-- ============================================
-- 8. 告警条件表 (alert_conditions)
-- ============================================
DROP TABLE IF EXISTS alert_conditions CASCADE;
CREATE TABLE alert_conditions (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT NOT NULL,
    field_type VARCHAR(50) NOT NULL,     -- status, response_time, uptime
    operator VARCHAR(10) NOT NULL,       -- eq, ne, gt, lt, ge, le, contains
    value TEXT,
    logical_op VARCHAR(5),               -- and, or
    "order" INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_alert_conditions_rule_id ON alert_conditions(rule_id);

-- 添加注释
COMMENT ON TABLE alert_conditions IS '告警条件表';
COMMENT ON COLUMN alert_conditions.field_type IS '字段类型: status, response_time, uptime';
COMMENT ON COLUMN alert_conditions.operator IS '操作符: eq, ne, gt, lt, ge, le, contains';

-- ============================================
-- 9. 告警规则组表 (alert_rule_groups)
-- ============================================
DROP TABLE IF EXISTS alert_rule_groups CASCADE;
CREATE TABLE alert_rule_groups (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT NOT NULL,
    name VARCHAR(255),
    logical_op VARCHAR(5),               -- and, or
    "order" INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX idx_alert_rule_groups_rule_id ON alert_rule_groups(rule_id);

-- 添加注释
COMMENT ON TABLE alert_rule_groups IS '告警规则组表';

-- ============================================
-- 10. 告警历史表 (alert_history)
-- ============================================
DROP TABLE IF EXISTS alert_history CASCADE;
CREATE TABLE alert_history (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT,
    target_id INTEGER,
    channel_id INTEGER,
    severity VARCHAR(50),
    status VARCHAR(50),
    message TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_alert_history_rule_id ON alert_history(rule_id);
CREATE INDEX idx_alert_history_target_id ON alert_history(target_id);
CREATE INDEX idx_alert_history_sent_at ON alert_history(sent_at);

-- 添加注释
COMMENT ON TABLE alert_history IS '告警历史表';

-- ============================================
-- 自动更新 updated_at 触发器函数
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要的表创建触发器
DROP TRIGGER IF EXISTS update_monitor_targets_updated_at ON monitor_targets;
CREATE TRIGGER update_monitor_targets_updated_at
    BEFORE UPDATE ON monitor_targets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_ip_geo_cache_updated_at ON ip_geo_cache;
CREATE TRIGGER update_ip_geo_cache_updated_at
    BEFORE UPDATE ON ip_geo_cache
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_dns_providers_updated_at ON dns_providers;
CREATE TRIGGER update_dns_providers_updated_at
    BEFORE UPDATE ON dns_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_alert_channels_updated_at ON alert_channels;
CREATE TRIGGER update_alert_channels_updated_at
    BEFORE UPDATE ON alert_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;
CREATE TRIGGER update_alert_rules_updated_at
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 初始化数据
-- ============================================

-- 插入默认 DNS 供应商
INSERT INTO dns_providers (name, server, server_type, is_default) VALUES
    ('Google DNS', '8.8.8.8:53', 'udp', true),
    ('Cloudflare DNS', '1.1.1.1:53', 'udp', false),
    ('阿里 DNS', '223.5.5.5:53', 'udp', false),
    ('腾讯 DNS', '119.29.29.29:53', 'udp', false),
    ('114 DNS', '114.114.114.114:53', 'udp', false)
ON CONFLICT DO NOTHING;

-- ============================================
-- 授权（如果需要）
-- ============================================
-- 授予用户权限（取消注释并修改用户名）
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO monitor_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO monitor_user;

-- ============================================
-- 分析表以优化查询
-- ============================================
ANALYZE monitor_targets;
ANALYZE monitor_status;
ANALYZE monitor_history;
ANALYZE ip_geo_cache;
ANALYZE dns_providers;
ANALYZE alert_channels;
ANALYZE alert_rules;
ANALYZE alert_conditions;
ANALYZE alert_rule_groups;
ANALYZE alert_history;

-- ============================================
-- 初始化完成
-- ============================================
-- 数据库初始化成功完成！
-- 表总数: 10
-- 索引: 已创建
-- 外键: 已创建
-- 触发器: 已创建（自动更新 updated_at）
-- 默认数据: 已插入
-- 字符集: UTF8
-- 时区支持: 已启用（TIMESTAMP WITH TIME ZONE）
