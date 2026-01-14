# ArrowGo监控拨测系统- 完整文档

**版本**: v0.1
**更新日期**: 2025-01-11
**状态**: ✅ 生产就绪

---

## 📑 目录

- [API接口文档](#api接口文档)
- [Web界面使用指南](#web界面使用指南)
- [系统配置详解](#系统配置详解)
- [部署运维指南](#部署运维指南)
- [故障排查指南](#故障排查指南)
- [功能特性详解](#功能特性详解)
- [版本更新日志](#版本更新日志)

---

## API接口文档

### 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **请求方式**: POST (所有接口)

---

### 监控管理接口

#### 1. 添加监控

**接口**: `POST /api/v1/monitor/add`

**请求参数**:
```json
{
  "name": "百度搜索",
  "type": "https",
  "address": "https://www.baidu.com",
  "port": 443,
  "interval": 60,
  "enabled": true,
  "http_method": "GET",
  "http_headers": {
    "User-Agent": "Mozilla/5.0...",
    "Accept": "*/*"
  },
  "http_body": "",
  "resolved_host": "",
  "follow_redirects": true,
  "max_redirects": 10,
  "expected_status_codes": [200, 301],
  "dns_server": "8.8.8.8",
  "ssl_warn_days": 30,
  "ssl_critical_days": 7,
  "ssl_check": true,
  "ssl_get_chain": true
}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 16,
    "name": "百度搜索",
    "status": "up",
    "response_time": 99,
    "uptime_percentage": 100
  }
}
```

---

#### 2. 列出监控

**接口**: `POST /api/v1/monitor/list`

**请求参数**:
```json
{}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 16,
      "name": "百度搜索",
      "type": "https",
      "address": "https://www.baidu.com",
      "interval": 60,
      "enabled": true
    }
  ]
}
```

---

#### 3. 获取监控详情

**接口**: `POST /api/v1/monitor/get`

**请求参数**:
```json
{
  "id": 16
}
```

**响应**: 返回完整的监控配置信息

---

#### 4. 更新监控

**接口**: `POST /api/v1/monitor/update`

**请求参数**: 与添加监控相同，需包含 `id` 字段

---

#### 5. 删除监控

**接口**: `POST /api/v1/monitor/remove`

**请求参数**:
```json
{
  "id": 16
}
```

**说明**: 会自动清理关联的历史记录和状态数据

---

### 监控状态接口

#### 1. 获取单个监控状态

**接口**: `POST /api/v1/monitor/status/get`

**请求参数**:
```json
{
  "target_id": 16
}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "target_id": 16,
    "status": "up",
    "response_time": 99,
    "message": "HTTP 200 OK",
    "resolved_ip": "110.242.68.66",
    "ssl_days_until_expiry": 311,
    "checked_at": "2026-01-14T15:20:26+08:00",
    "uptime_percentage": 100
  }
}
```

---

#### 2. 列出所有监控状态

**接口**: `POST /api/v1/monitor/status/list`

**请求参数**:
```json
{}
```

---

### 日志查询接口

#### 1. 查询日志（文件存储）

**接口**: `POST /api/v1/logs/query`

**请求参数**:
```json
{
  "target_id": 16,
  "status": "up",
  "start_time": "2026-01-14T00:00:00+08:00",
  "end_time": "2026-01-14T23:59:59+08:00",
  "page": 1,
  "page_size": 20
}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "logs": [
      {
        "timestamp": "2026-01-14T15:20:26+08:00",
        "target_id": 16,
        "target_name": "百度搜索",
        "type": "https",
        "address": "https://www.baidu.com",
        "status": "up",
        "response_time": 85,
        "message": "HTTP 200 OK",
        "request": {
          "method": "GET",
          "url": "https://www.baidu.com",
          "headers": {
            "User-Agent": "Mozilla/5.0...",
            "Accept": "*/*",
            "Accept-Encoding": "gzip, deflate, br",
            "Accept-Language": "zh-CN,zh;q=0.9",
            "Connection": "keep-alive"
          }
        },
        "response": {
          "status_code": 200,
          "body_size": 10845,
          "headers": {
            "Content-Type": "text/html",
            "Server": "Apache",
            "title": "百度一下，你就知道",
            "resolved_ip": "110.242.68.66",
            "days_until_expiry": "311"
          }
        }
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 2. 搜索日志（Elasticsearch）

**接口**: `POST /api/v1/logs/search`

**说明**: 需要启用Elasticsearch配置

---

#### 3. 获取日志统计

**接口**: `POST /api/v1/logs/stats`

**说明**: 返回监控成功率、平均响应时间等统计数据

---

### IP查询接口

#### IP地理位置查询

**接口**: `POST /api/v1/ipgeo/query`

**请求参数**:
```json
{
  "ip": "8.8.8.8"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "ip": "8.8.8.8",
    "country": "美国",
    "region": "加利福尼亚州",
    "city": "山景城",
    "isp": "Google LLC",
    "latitude": 37.422,
    "longitude": -122.084
  }
}
```

---

## Web界面使用指南

### 监控管理页面

#### 添加监控

1. 点击"添加监控"按钮
2. 填写基本信息：
   - **名称**: 监控目标的友好名称
   - **类型**: 选择HTTP/HTTPS/TCP/UDP/DNS
   - **地址**: 支持完整URL（如 `https://example.com/path`）
   - **端口**: 自动检测或手动指定
3. 配置高级选项：
   - **HTTP方法**: GET/POST/PUT/DELETE等
   - **请求头**: 从预设选择或自定义
   - **请求体**: POST请求的JSON数据
   - **Host头**: 自定义主机名
   - **DNS服务器**: 指定DNS解析服务器
   - **重定向设置**: 是否跟随重定向
   - **期望状态码**: 定义成功的HTTP状态码
4. SSL证书监控（HTTPS类型）：
   - 勾选"SSL证书监控"
   - 设置告警天数（警告/严重）
   - 勾选"获取证书链"
5. 设置检查间隔（秒）
6. 点击"保存"并立即触发检查

---

#### 查看监控详情

点击监控列表中的"查看详情"按钮，显示：

**基本信息**:
- 监控名称、类型、地址
- 当前状态（在线/离线）
- 响应时间
- 正常运行时间百分比

**域名详情**:
- 域名
- DNS解析的IP地址

**SSL证书信息**（HTTPS类型）:
- 完整证书链
  - 终端证书（服务器证书）
  - 中间证书
  - 根证书
- 每个证书的详细信息：
  - 主体（Subject）
  - 颁发者（Issuer）
  - 序列号
  - 生效日期
  - 过期日期
  - 剩余天数
  - 指纹

**最近拨测日志（最近20条）**:
- 检查时间
- 状态
- 响应时间
- 消息
- 点击"详情"按钮查看完整请求/响应

---

#### 编辑/删除监控

- **编辑**: 点击"编辑"按钮，修改配置后保存
- **删除**: 点击"删除"按钮，确认后删除（自动清理关联数据）

---

### 日志查询页面

#### 搜索条件

- **目标**: 选择特定监控目标
- **状态**: 选择up/down
- **时间范围**: 指定开始和结束时间
- **分页**: 每页显示20条，支持翻页

#### 查看日志详情

点击日志列表中的"查看详情"，显示：
- 基本信息部分
- 检查消息
- 请求详情（方法、URL、请求头、请求体）
- 响应详情（状态码、大小、响应头）

---

### IP查询页面

输入IP地址，查询：
- 国家/地区/城市
- ISP运营商
- 经纬度坐标

---

## 系统配置详解

### config.yaml 完整配置

```yaml
# 服务器配置
server:
  http_port: 8080              # HTTP服务端口
  grpc_port: 9090              # gRPC服务端口
  host: 0.0.0.0                # 监听地址

# 数据库配置
database:
  driver: sqlite               # 数据库类型: sqlite/mysql/postgres
  dbname: monitor.db           # 数据库名称
  host: localhost              # 数据库主机（MySQL/PostgreSQL）
  port: 3306                   # 数据库端口
  username: root               # 数据库用户名
  password: ""                 # 数据库密码
  charset: utf8mb4            # 字符集

# 监控配置
monitor:
  check_interval: 60           # 默认检查间隔（秒）
  workers: 100                 # 并发工作线程数
  timeout: 30                  # 请求超时时间（秒）

# 日志配置
logger:
  level: info                  # 日志级别: debug/info/warn/error
  output: stdout               # 输出: stdout/file
  file_path: logs/monitor.log  # 日志文件路径

# Elasticsearch配置（可选）
elasticsearch:
  enabled: false               # 是否启用ES
  addresses:
    - http://localhost:9200
  username: ""                 # ES用户名
  password: ""                 # ES密码
  index_prefix: monitor        # 索引前缀

# 告警配置（开发中）
alert:
  enabled: false               # 是否启用告警
  channels: []                 # 告警通道配置
```

---

### 环境变量

也可以通过环境变量配置：

```bash
export MONITOR_HTTP_PORT=8080
export MONITOR_DB_DRIVER=sqlite
export MONITOR_DB_NAME=monitor.db
export MONITOR_LOG_LEVEL=info
export MONITOR_ES_ENABLED=false
```

---

## 部署运维指南

### 开发环境

```bash
# 1. 克隆项目
git clone <repository-url>
cd monitor

# 2. 安装依赖
go mod download

# 3. 运行
go run cmd/server/main.go

# 4. 访问
open http://localhost:8080
```

---

### 生产环境

#### 方式1: 直接运行

```bash
# 编译
go build -o monitor cmd/server/main.go

# 运行
./monitor
```

#### 方式2: Docker部署

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o monitor cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/monitor .
COPY --from=builder /app/web ./web
COPY --from=builder /app/config.yaml .

EXPOSE 8080
CMD ["./monitor"]
```

构建和运行：
```bash
docker build -t monitor:latest .
docker run -d -p 8080:8080 -v $(pwd)/data:/root/data monitor:latest
```

---

#### 方式3: Systemd服务

创建 `/etc/systemd/system/monitor.service`:

```ini
[Unit]
Description=Monitor Service
After=network.target

[Service]
Type=simple
User=monitor
WorkingDirectory=/opt/monitor
ExecStart=/opt/monitor/monitor
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable monitor
sudo systemctl start monitor
sudo systemctl status monitor
```

---

### Nginx反向代理

```nginx
server {
    listen 80;
    server_name monitor.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

### 数据库管理

#### SQLite

```bash
# 备份数据库
cp monitor.db monitor.db.backup

# 查看数据
sqlite3 monitor.db "SELECT * FROM monitor_targets;"
```

#### MySQL/PostgreSQL

```bash
# 备份
mysqldump -u root -p monitor > monitor_backup.sql

# 恢复
mysql -u root -p monitor < monitor_backup.sql
```

---

### 日志管理

#### 文件日志

日志存储在 `logs/` 目录：
- `check-YYYY-MM-DD.jsonl` - 检查日志
- `monitor.log` - 系统日志

查看日志：
```bash
# 查看今天的检查日志
cat logs/check-$(date +%Y-%m-%d).jsonl | jq

# 查找失败的检查
grep '"status":"down"' logs/check-*.jsonl
```

#### Elasticsearch日志

配置Kibana查看ES日志：
- Index Pattern: `monitor-*`
- Time Field: `timestamp`

---

### 性能优化

1. **调整并发数**:
   ```yaml
   monitor:
     workers: 200  # 增加到200个并发worker
   ```

2. **启用Elasticsearch**:
   ```yaml
   elasticsearch:
     enabled: true
   ```

3. **数据库优化**:
   - 使用MySQL/PostgreSQL替代SQLite
   - 定期清理历史数据

---

## 故障排查指南

### 常见问题

#### 1. 监控一直显示down

**可能原因**:
- 网络不通
- 端口错误
- 证书过期
- DNS解析失败

**排查步骤**:
```bash
# 测试网络连通性
curl -I https://example.com

# 检查DNS解析
nslookup example.com

# 查看日志
tail -f logs/monitor.log
```

---

#### 2. SSL证书检查失败

**可能原因**:
- 证书已过期
- 证书链不完整
- 时间不同步

**解决方案**:
```bash
# 查看证书详情
openssl s_client -connect example.com:443 -showcerts

# 同步时间
ntpdate pool.ntp.org
```

---

#### 3. 日志无法查询

**文件日志**:
```bash
# 检查日志文件
ls -lh logs/

# 查看日志格式
head -n 1 logs/check-2026-01-14.jsonl | jq
```

**Elasticsearch**:
```bash
# 检查ES连接
curl http://localhost:9200/_cluster/health

# 查看索引
curl http://localhost:9200/_cat/indices?v
```

---

#### 4. 性能问题

**检查系统资源**:
```bash
# CPU和内存
top

# 磁盘IO
iostat -x 1

# 网络连接
netstat -an | grep ESTABLISHED | wc -l
```

**优化配置**:
- 降低检查间隔
- 减少并发worker数
- 清理历史数据

---

#### 5. Web界面无法访问

**检查服务**:
```bash
# 查看进程
ps aux | grep monitor

# 查看端口
lsof -i :8080

# 测试API
curl http://localhost:8080/api/v1/monitor/list
```

---

### 日志级别调整

开发调试时启用debug级别：
```yaml
logger:
  level: debug
```

生产环境使用info或warn级别：
```yaml
logger:
  level: warn
```

---

## 功能特性详解

### SSL/TLS证书监控

系统自动获取完整证书链，包括：

**终端实体证书**（服务器证书）:
- 主体CN（通用名称）
- 颁发者
- 序列号
- 生效日期
- 过期日期
- 剩余天数
- SAN（Subject Alternative Names）

**中间证书**:
- 连接终端证书和根证书
- 同样显示详细信息

**根证书**:
- 信任锚点
- 通常是知名CA机构

---

### DNS解析监控

支持自定义DNS服务器：
- Google DNS: `8.8.8.8`
- Cloudflare DNS: `1.1.1.1`
- 阿里DNS: `223.5.5.5`
- 自定义DNS服务器

解析结果：
- IPv4地址（优先）
- IPv6地址
- 保存到日志 `resolved_ip` 字段

---

### HTTP请求头预设

系统自动添加默认请求头：
```
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) ...
Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: zh-CN,zh;q=0.9,en;q=0.8
Connection: keep-alive
```

用户可自定义或覆盖默认值。

---

### 文件日志格式

JSONL格式（每行一个JSON对象）:

```json
{
  "timestamp": "2026-01-14T15:20:26+08:00",
  "target_id": 16,
  "target_name": "百度搜索",
  "type": "https",
  "address": "https://www.baidu.com",
  "status": "up",
  "response_time": 85,
  "message": "HTTP 200 OK",
  "request": {
    "method": "GET",
    "url": "https://www.baidu.com",
    "headers": {...}
  },
  "response": {
    "status_code": 200,
    "body_size": 10845,
    "headers": {
      "title": "百度一下，你就知道",
      "resolved_ip": "110.242.68.66",
      "days_until_expiry": "311"
    }
  }
}
```

---

### 并发检查架构

系统使用Worker Pool模式：
- 100个并发worker（可配置）
- 检查队列缓冲区（1000容量）
- 非阻塞ES写入（500缓冲区）
- 自动负载均衡

---

## 版本更新日志

### v0.1 (2025-01-11)

**新增功能**:
- ✅ 完整SSL/TLS证书链获取
- ✅ DNS解析真实IP地址
- ✅ 文件日志存储（JSONL格式）
- ✅ 完整请求/响应头保存
- ✅ HTML页面标题提取
- ✅ 默认HTTP请求头
- ✅ 拨测日志详情查看功能
- ✅ 请求头/响应头完整显示

**优化**:
- 🎨 优化UI性能
- ⚡ 60秒自动刷新
- 🔧 改进错误处理
- 📊 增强日志详情展示

**修复**:
- 🐛 修复日志详情模态框缺失问题
- 🐛 修复请求头保存不完整问题
- 🐛 修复响应体大小字段不匹配问题

---

### v0.09 (2025-01-10)

**新增功能**:
- ✅ Web管理界面
- ✅ 文件日志查询
- ✅ IP地理位置查询
- ✅ 实时状态显示

---

### v0.08 (2025-01-09)

**初始版本**:
- ✅ 多协议监控
- ✅ HTTP/HTTPS/TCP/UDP/DNS支持
- ✅ Elasticsearch集成
- ✅ 告警引擎框架

---

## 附录

### 支持的监控类型

| 类型 | 说明 | 端口 | 特性 |
|------|------|------|------|
| HTTP | HTTP协议检查 | 80 | 自定义方法/头/体 |
| HTTPS | HTTPS协议检查 | 443 | SSL证书监控 |
| TCP | TCP端口检查 | 自定义 | 连通性检查 |
| UDP | UDP端口检查 | 自定义 | 连通性检查 |
| DNS | DNS解析检查 | 53 | 自定义DNS服务器 |

---

### 退出状态码

| 代码 | 含义 |
|------|------|
| 0 | 正常退出 |
| 1 | 配置错误 |
| 2 | 数据库连接失败 |
| 3 | 端口占用 |

--

### 联系方式

- 问题反馈: GitHub Issues
- 文档: `DOCUMENTATION.md`
- 许可证: Apache License 2.0

---

<div align="center">

**⬆ 返回顶部**

**Made with ❤️**


</div>
