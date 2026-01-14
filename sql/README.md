# 数据库初始化脚本

本目录包含 ArrowGo 监控拨测系统的数据库初始化脚本，支持三种主流数据库。

---

## 📁 文件列表

| 文件名 | 数据库 | 版本要求 | 说明 |
|--------|--------|----------|------|
| `sqlite_init.sql` | SQLite 3 | 3.0+ | 轻量级，适合开发和小型部署 |
| `mysql_init.sql` | MySQL | 5.7+ | 企业级关系数据库 |
| `pgsql_init.sql` | PostgreSQL | 12+ | 高级开源关系数据库 |

---

## 🚀 快速开始

### 1. SQLite 初始化

**最简单的方式，无需额外安装**

```bash
# 进入项目目录
cd /path/to/ArrowGo

# 创建数据库目录
mkdir -p data

# 初始化数据库
sqlite3 data/monitor.db < sql/sqlite_init.sql

# 验证
sqlite3 data/monitor.db "SELECT COUNT(*) FROM dns_providers;"
# 应该返回: 5
```

**配置文件 (config.yaml)**:
```yaml
database:
  driver: sqlite
  dbname: data/monitor.db
```

---

### 2. MySQL 初始化

**适合生产环境**

```bash
# 1. 登录 MySQL
mysql -u root -p

# 2. 创建数据库和用户
CREATE DATABASE monitor CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'monitor_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON monitor.* TO 'monitor_user'@'localhost';
FLUSH PRIVILEGES;
EXIT;

# 3. 导入初始化脚本
mysql -u root -p monitor < sql/mysql_init.sql

# 4. 验证
mysql -u root -p monitor -e "SELECT COUNT(*) FROM dns_providers;"
# 应该返回: 5
```

**配置文件 (config.yaml)**:
```yaml
database:
  driver: mysql
  host: localhost
  port: 3306
  username: monitor_user
  password: your_password
  dbname: monitor
  charset: utf8mb4
```

---

### 3. PostgreSQL 初始化

**适合需要高级功能的场景**

```bash
# 1. 登录 PostgreSQL
sudo -u postgres psql

# 2. 创建数据库和用户
CREATE DATABASE monitor ENCODING 'UTF8';
CREATE USER monitor_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE monitor TO monitor_user;
EXIT;

# 3. 导入初始化脚本
psql -U postgres -d monitor -f sql/pgsql_init.sql

# 4. 验证
psql -U postgres -d monitor -c "SELECT COUNT(*) FROM dns_providers;"
# 应该返回: 5
```

**配置文件 (config.yaml)**:
```yaml
database:
  driver: postgres
  host: localhost
  port: 5432
  username: monitor_user
  password: your_password
  dbname: monitor
  sslmode: disable
```

---

## 📊 数据库表结构

所有数据库都包含以下10张表：

| 表名 | 说明 | 主要字段 |
|------|------|----------|
| `monitor_targets` | 监控目标配置 | name, type, address, interval, enabled |
| `monitor_status` | 当前监控状态 | target_id, status, response_time |
| `monitor_history` | 历史监控记录 | target_id, status, checked_at |
| `ip_geo_cache` | IP地理位置缓存 | ip, country, city, isp |
| `dns_providers` | DNS供应商 | name, server, server_type |
| `alert_channels` | 告警渠道 | name, type, config |
| `alert_rules` | 告警规则 | target_id, channel_id, threshold |
| `alert_conditions` | 告警条件 | rule_id, field_type, operator |
| `alert_rule_groups` | 告警规则组 | rule_id, name, logical_op |
| `alert_history` | 告警历史 | rule_id, severity, message |

---

## 🔧 维护操作

### SQLite

```bash
# 备份
cp data/monitor.db data/monitor.db.backup.$(date +%Y%m%d)

# 查看表结构
sqlite3 data/monitor.db ".schema monitor_targets"

# 查看数据
sqlite3 data/monitor.db "SELECT * FROM monitor_targets LIMIT 10;"

# 清理历史数据（保留30天）
sqlite3 data/monitor.db "DELETE FROM monitor_history WHERE checked_at < datetime('now', '-30 days');"

# 优化数据库
sqlite3 data/monitor.db "VACUUM;"
```

### MySQL

```bash
# 备份
mysqldump -u root -p monitor > backup_$(date +%Y%m%d).sql

# 恢复
mysql -u root -p monitor < backup_20250114.sql

# 查看表结构
mysql -u root -p monitor -e "DESCRIBE monitor_targets;"

# 清理历史数据（保留30天）
mysql -u root -p monitor -e "DELETE FROM monitor_history WHERE checked_at < DATE_SUB(NOW(), INTERVAL 30 DAY);"

# 优化表
mysql -u root -p monitor -e "OPTIMIZE TABLE monitor_history;"

# 查看表大小
mysql -u root -p monitor -e "
    SELECT
        table_name,
        ROUND(((data_length + index_length) / 1024 / 1024), 2) AS 'Size (MB)'
    FROM information_schema.TABLES
    WHERE table_schema = 'monitor'
    ORDER BY (data_length + index_length) DESC;
"
```

### PostgreSQL

```bash
# 备份
pg_dump -U postgres monitor > backup_$(date +%Y%m%d).sql

# 恢复
psql -U postgres monitor < backup_20250114.sql

# 查看表结构
psql -U postgres -d monitor -c "\d monitor_targets"

# 清理历史数据（保留30天）
psql -U postgres -d monitor -c "DELETE FROM monitor_history WHERE checked_at < NOW() - INTERVAL '30 days';"

# 优化表（VACUUM ANALYZE）
psql -U postgres -d monitor -c "VACUUM ANALYZE monitor_history;"

# 查看表大小
psql -U postgres -d monitor -c "
    SELECT
        table_name,
        pg_size_pretty(pg_total_relation_size(quote_ident(table_name))) AS size
    FROM information_schema.tables
    WHERE table_schema = 'public'
    ORDER BY pg_total_relation_size(quote_ident(table_name)) DESC;
"
```

---

## 🔄 数据库迁移

### 从 SQLite 迁移到 MySQL

```bash
# 1. 导出 SQLite 数据
sqlite3 data/monitor.db .dump > dump.sql

# 2. 转换 SQL 语法（需要手动调整）
# - 将 AUTOINCREMENT 改为 AUTO_INCREMENT
# - 将 INTEGER 改为对应的 MySQL 类型
# - 调整日期时间函数

# 3. 导入到 MySQL
mysql -u root -p monitor < dump.sql
```

### 从 MySQL 迁移到 PostgreSQL

```bash
# 1. 导出 MySQL 数据
mysqldump -u root -p monitor > mysql_dump.sql

# 2. 转换 SQL 语法（需要手动调整）
# - 将 AUTO_INCREMENT 改为 SERIAL
# - 将 TINYINT(1) 改为 BOOLEAN
# - 调整日期时间函数

# 3. 导入到 PostgreSQL
psql -U postgres -d monitor < mysql_dump.sql
```

---

## ⚙️ 性能优化建议

### SQLite

- 定期执行 `VACUUM` 回收空间
- 考虑使用 WAL 模式：`PRAGMA journal_mode=WAL;`
- 增加缓存大小：`PRAGMA cache_size=-64000;` (64MB)

### MySQL

- 为大表添加适当的索引
- 调整 `innodb_buffer_pool_size`（建议为物理内存的70-80%）
- 定期执行 `OPTIMIZE TABLE`
- 考虑分区表（按时间分区 monitor_history）

### PostgreSQL

- 调整 `shared_buffers`（建议为物理内存的25%）
- 调整 `effective_cache_size`（建议为物理内存的50-75%）
- 定期执行 `VACUUM ANALYZE`
- 考虑表分区（pg_partition 扩展）

---

## 🔒 安全建议

1. **使用强密码**: 为数据库用户设置复杂密码
2. **限制访问**: 只允许本地访问或使用防火墙
3. **定期备份**: 每天自动备份数据库
4. **监控日志**: 启用慢查询日志
5. **更新权限**: 定期审查用户权限

### MySQL 安全设置

```sql
-- 删除匿名用户
DELETE FROM mysql.user WHERE User='';

-- 禁止 root 远程登录
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');

-- 刷新权限
FLUSH PRIVILEGES;
```

### PostgreSQL 安全设置

```sql
-- 修改 postgres 用户密码
ALTER USER postgres WITH PASSWORD 'strong_password';

-- 编辑 pg_hba.conf 设置认证方式
# 建议: 将 peer 改为 md5 或 scram-sha-256
```

---

## 🐛 常见问题

### Q: SQLite 提示 "database is locked"

**A**: SQLite 不支持高并发写操作，建议：
- 使用 MySQL 或 PostgreSQL
- 或增加 busy_timeout：`PRAGMA busy_timeout=5000;`

### Q: MySQL 字符集问题

**A**: 确保使用 utf8mb4：
```sql
ALTER DATABASE monitor CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### Q: PostgreSQL 时区问题

**A**: 使用 TIMESTAMP WITH TIME ZONE 类型存储时间

### Q: 数据库连接失败

**A**: 检查：
1. 数据库服务是否运行
2. 用户名密码是否正确
3. 防火墙是否允许连接
4. config.yaml 配置是否正确

---

##  获取帮助

- 查看完整文档: [DOCUMENTATION.md](../DOCUMENTATION.md)
- 问题反馈: GitHub Issues
- 官方文档:
  - SQLite: https://www.sqlite.org/docs.html
  - MySQL: https://dev.mysql.com/doc/
  - PostgreSQL: https://www.postgresql.org/docs/

---

<div align="center">

**Made with ❤️**



</div>
