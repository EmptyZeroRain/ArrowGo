-- ============================================
-- Monitor 监控拨测系统 - MySQL 初始化脚本
-- 版本: v3.0
-- 更新日期: 2025-01-14
-- 兼容: MySQL 5.7+, MariaDB 10.2+
-- ============================================

-- 设置字符集和排序规则
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 1. 监控目标表 (monitor_targets)
-- ============================================
DROP TABLE IF EXISTS `monitor_targets`;
CREATE TABLE `monitor_targets` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL COMMENT '监控目标名称',
    `type` VARCHAR(50) NOT NULL COMMENT '类型: http, https, tcp, udp, dns',
    `address` VARCHAR(500) NOT NULL COMMENT '监控地址',
    `port` INT DEFAULT NULL COMMENT '端口号',
    `interval` BIGINT DEFAULT 60 COMMENT '检查间隔（秒）',
    `metadata` TEXT COMMENT '元数据（JSON）',
    `enabled` TINYINT(1) DEFAULT 1 COMMENT '是否启用',

    -- HTTP/HTTPS 专用字段
    `http_method` VARCHAR(10) DEFAULT NULL COMMENT 'HTTP方法: GET, POST, PUT, DELETE',
    `http_headers` TEXT COMMENT 'HTTP请求头（JSON）',
    `http_body` TEXT COMMENT 'HTTP请求体',
    `resolved_host` VARCHAR(255) DEFAULT NULL COMMENT '自定义Host头',
    `follow_redirects` TINYINT(1) DEFAULT 1 COMMENT '是否跟随重定向',
    `max_redirects` INT DEFAULT 10 COMMENT '最大重定向次数',
    `expected_status_codes` TEXT COMMENT '期望的状态码（逗号分隔）',

    -- DNS 专用字段
    `dns_server` VARCHAR(255) DEFAULT NULL COMMENT 'DNS服务器地址',
    `dns_server_name` VARCHAR(255) DEFAULT NULL COMMENT 'DNS服务器名称',
    `dns_server_type` VARCHAR(10) DEFAULT NULL COMMENT 'DNS协议: udp, tcp, doh, dot',

    -- PING 专用字段
    `ping_count` INT DEFAULT 4 COMMENT 'PING次数',
    `ping_size` INT DEFAULT 32 COMMENT 'PING包大小（字节）',
    `ping_timeout` INT DEFAULT 5000 COMMENT 'PING超时（毫秒）',

    -- SMTP 专用字段
    `smtp_username` VARCHAR(255) DEFAULT NULL COMMENT 'SMTP用户名',
    `smtp_password` VARCHAR(255) DEFAULT NULL COMMENT 'SMTP密码',
    `smtp_use_tls` TINYINT(1) DEFAULT 0 COMMENT '是否使用TLS',
    `smtp_mail_from` VARCHAR(255) DEFAULT NULL COMMENT '测试邮件发件人',
    `smtp_mail_to` VARCHAR(255) DEFAULT NULL COMMENT '测试邮件收件人',
    `smtp_check_starttls` TINYINT(1) DEFAULT 1 COMMENT '检查STARTTLS支持',

    -- SNMP 专用字段
    `snmp_community` VARCHAR(255) DEFAULT NULL COMMENT 'SNMP Community字符串',
    `snmp_oid` VARCHAR(500) DEFAULT NULL COMMENT 'SNMP OID',
    `snmp_version` VARCHAR(10) DEFAULT NULL COMMENT 'SNMP版本: v1, v2c, v3',
    `snmp_expected_value` VARCHAR(255) DEFAULT NULL COMMENT '期望值',
    `snmp_operator` VARCHAR(10) DEFAULT NULL COMMENT '操作符: eq, ne, gt, lt',

    -- SSL/TLS 证书专用字段
    `ssl_warn_days` INT DEFAULT 30 COMMENT 'SSL证书警告天数',
    `ssl_critical_days` INT DEFAULT 7 COMMENT 'SSL证书严重天数',
    `ssl_get_chain` TINYINT(1) DEFAULT 1 COMMENT '获取证书链',
    `ssl_check` TINYINT(1) DEFAULT 0 COMMENT '启用SSL证书监控',

    -- 告警渠道关联
    `alert_channel_ids` TEXT COMMENT '告警渠道ID列表（JSON数组）',

    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    KEY `idx_type` (`type`),
    KEY `idx_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='监控目标表';

-- ============================================
-- 2. 监控状态表 (monitor_status)
-- ============================================
DROP TABLE IF EXISTS `monitor_status`;
CREATE TABLE `monitor_status` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `target_id` INT UNSIGNED NOT NULL,
    `status` VARCHAR(50) NOT NULL COMMENT '状态: up, down, unknown',
    `response_time` BIGINT DEFAULT NULL COMMENT '响应时间（毫秒）',
    `message` TEXT COMMENT '状态消息',
    `checked_at` TIMESTAMP NULL DEFAULT NULL COMMENT '检查时间',
    `uptime_percentage` INT DEFAULT 0 COMMENT '正常运行时间百分比',

    -- SSL 证书信息
    `ssl_days_until_expiry` INT DEFAULT NULL COMMENT 'SSL证书剩余天数',
    `ssl_issuer` VARCHAR(255) DEFAULT NULL COMMENT 'SSL证书颁发者',
    `ssl_subject` VARCHAR(255) DEFAULT NULL COMMENT 'SSL证书主题',
    `ssl_serial` VARCHAR(128) DEFAULT NULL COMMENT 'SSL证书序列号',

    -- DNS 信息
    `dns_records` TEXT COMMENT 'DNS记录（JSON）',
    `resolved_ip` VARCHAR(64) DEFAULT NULL COMMENT '解析的IP地址',

    -- 额外的检查数据
    `data` TEXT COMMENT '完整检查结果数据（JSON）',

    PRIMARY KEY (`id`),
    KEY `idx_target_id` (`target_id`),
    KEY `idx_checked_at` (`checked_at`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_monitor_status_target` FOREIGN KEY (`target_id`) REFERENCES `monitor_targets` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='监控状态表';

-- ============================================
-- 3. 监控历史表 (monitor_history)
-- ============================================
DROP TABLE IF EXISTS `monitor_history`;
CREATE TABLE `monitor_history` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `target_id` INT UNSIGNED NOT NULL,
    `status` VARCHAR(50) NOT NULL COMMENT '状态: up, down',
    `response_time` BIGINT DEFAULT NULL COMMENT '响应时间（毫秒）',
    `message` TEXT COMMENT '消息',
    `checked_at` TIMESTAMP NULL DEFAULT NULL COMMENT '检查时间',
    PRIMARY KEY (`id`),
    KEY `idx_target_id` (`target_id`),
    KEY `idx_checked_at` (`checked_at`),
    CONSTRAINT `fk_monitor_history_target` FOREIGN KEY (`target_id`) REFERENCES `monitor_targets` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='监控历史表';

-- ============================================
-- 4. IP 地理位置缓存表 (ip_geo_cache)
-- ============================================
DROP TABLE IF EXISTS `ip_geo_cache`;
CREATE TABLE `ip_geo_cache` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `ip` VARCHAR(45) NOT NULL COMMENT 'IP地址（IPv4/IPv6）',
    `country` VARCHAR(100) DEFAULT NULL COMMENT '国家',
    `region` VARCHAR(100) DEFAULT NULL COMMENT '地区/省份',
    `city` VARCHAR(100) DEFAULT NULL COMMENT '城市',
    `isp` VARCHAR(255) DEFAULT NULL COMMENT 'ISP运营商',
    `latitude` DOUBLE DEFAULT NULL COMMENT '纬度',
    `longitude` DOUBLE DEFAULT NULL COMMENT '经度',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ip` (`ip`),
    KEY `idx_ip` (`ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='IP地理位置缓存表';

-- ============================================
-- 5. DNS 供应商表 (dns_providers)
-- ============================================
DROP TABLE IF EXISTS `dns_providers`;
CREATE TABLE `dns_providers` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL COMMENT 'DNS供应商名称',
    `server` VARCHAR(500) NOT NULL COMMENT 'DNS服务器地址',
    `server_type` VARCHAR(10) NOT NULL COMMENT 'DNS协议: udp, tcp, doh, dot',
    `is_default` TINYINT(1) DEFAULT 0 COMMENT '是否默认',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='DNS供应商表';

-- ============================================
-- 6. 告警渠道表 (alert_channels)
-- ============================================
DROP TABLE IF EXISTS `alert_channels`;
CREATE TABLE `alert_channels` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(255) NOT NULL COMMENT '渠道名称',
    `type` VARCHAR(50) NOT NULL COMMENT '渠道类型: email, webhook, dingtalk, wechat',
    `enabled` TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    `config` TEXT NOT NULL COMMENT '渠道配置（JSON）',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警渠道表';

-- ============================================
-- 7. 告警规则表 (alert_rules)
-- ============================================
DROP TABLE IF EXISTS `alert_rules`;
CREATE TABLE `alert_rules` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `target_id` INT UNSIGNED NOT NULL COMMENT '监控目标ID',
    `channel_id` INT UNSIGNED NOT NULL COMMENT '告警渠道ID',
    `threshold_type` VARCHAR(20) DEFAULT NULL COMMENT '阈值类型: failure_count, response_time',
    `threshold_value` INT DEFAULT NULL COMMENT '阈值',
    `enabled` TINYINT(1) DEFAULT 1 COMMENT '是否启用',

    -- 高级字段
    `condition_logic` TEXT COMMENT '条件逻辑（JSON）',
    `cooldown_seconds` INT DEFAULT 300 COMMENT '冷却时间（秒）',
    `last_alert_time` TIMESTAMP NULL DEFAULT NULL COMMENT '最后告警时间',

    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    KEY `idx_target_id` (`target_id`),
    KEY `idx_channel_id` (`channel_id`),
    CONSTRAINT `fk_alert_rules_target` FOREIGN KEY (`target_id`) REFERENCES `monitor_targets` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_alert_rules_channel` FOREIGN KEY (`channel_id`) REFERENCES `alert_channels` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警规则表';

-- ============================================
-- 8. 告警条件表 (alert_conditions)
-- ============================================
DROP TABLE IF EXISTS `alert_conditions`;
CREATE TABLE `alert_conditions` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `rule_id` BIGINT UNSIGNED NOT NULL COMMENT '规则ID',
    `field_type` VARCHAR(50) NOT NULL COMMENT '字段类型: status, response_time, uptime',
    `operator` VARCHAR(10) NOT NULL COMMENT '操作符: eq, ne, gt, lt, ge, le, contains',
    `value` TEXT COMMENT '阈值',
    `logical_op` VARCHAR(5) DEFAULT NULL COMMENT '逻辑操作符: and, or',
    `order` INT DEFAULT 0 COMMENT '执行顺序',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rule_id` (`rule_id`),
    CONSTRAINT `fk_alert_conditions_rule` FOREIGN KEY (`rule_id`) REFERENCES `alert_rules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警条件表';

-- ============================================
-- 9. 告警规则组表 (alert_rule_groups)
-- ============================================
DROP TABLE IF EXISTS `alert_rule_groups`;
CREATE TABLE `alert_rule_groups` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `rule_id` BIGINT UNSIGNED NOT NULL COMMENT '规则ID',
    `name` VARCHAR(255) DEFAULT NULL COMMENT '组名称',
    `logical_op` VARCHAR(5) DEFAULT NULL COMMENT '逻辑操作符: and, or',
    `order` INT DEFAULT 0 COMMENT '执行顺序',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rule_id` (`rule_id`),
    CONSTRAINT `fk_alert_rule_groups_rule` FOREIGN KEY (`rule_id`) REFERENCES `alert_rules` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警规则组表';

-- ============================================
-- 10. 告警历史表 (alert_history)
-- ============================================
DROP TABLE IF EXISTS `alert_history`;
CREATE TABLE `alert_history` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `rule_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '规则ID',
    `target_id` INT UNSIGNED DEFAULT NULL COMMENT '目标ID',
    `channel_id` INT UNSIGNED DEFAULT NULL COMMENT '渠道ID',
    `severity` VARCHAR(50) DEFAULT NULL COMMENT '严重程度',
    `status` VARCHAR(50) DEFAULT NULL COMMENT '状态',
    `message` TEXT COMMENT '消息',
    `sent_at` TIMESTAMP NULL DEFAULT NULL COMMENT '发送时间',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rule_id` (`rule_id`),
    KEY `idx_target_id` (`target_id`),
    KEY `idx_sent_at` (`sent_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='告警历史表';

-- ============================================
-- 初始化数据
-- ============================================

-- 插入默认 DNS 供应商
INSERT INTO `dns_providers` (`name`, `server`, `server_type`, `is_default`) VALUES
    ('Google DNS', '8.8.8.8:53', 'udp', 1),
    ('Cloudflare DNS', '1.1.1.1:53', 'udp', 0),
    ('阿里 DNS', '223.5.5.5:53', 'udp', 0),
    ('腾讯 DNS', '119.29.29.29:53', 'udp', 0),
    ('114 DNS', '114.114.114.114:53', 'udp', 0)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- ============================================
-- 恢复外键检查
-- ============================================
SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 初始化完成
-- ============================================
-- 数据库初始化成功完成！
-- 表总数: 10
-- 索引: 已创建
-- 外键: 已创建
-- 默认数据: 已插入
-- 字符集: utf8mb4
-- 排序规则: utf8mb4_unicode_ci
