<div align="center">

# ArrowGo监控拨测系统

</div>

<div align="center">

## 🎉 功能完整的监控系统

<br/>

![GitHub commit activity](https://img.shields.io/github/commit-activity/m/EmptyZeroRain/ArrowGo)
![GitHub last commit](https://img.shields.io/github/last-commit/EmptyZeroRain/ArrowGo)
![GitHub stars](https://img.shields.io/github/stars/EmptyZeroRain/ArrowGo)
![GitHub issues](https://img.shields.io/github/issues/EmptyZeroRain/ArrowGo)

<br/>

**多协议监控** | **Web管理界面** | **日志查询** | **智能告警**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://golang.org)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

</div>

---

## ✨ 特性亮点

- 🌐 **多协议支持** - HTTP/HTTPS/TCP/UDP/DNS
- 🔒 **SSL/TLS证书监控** - 获取证书链信息，监控过期时间
- 🖥️ **Web管理界面** - 可视化管理，操作简单
- 📊 **日志存储** - 文件日志 + Elasticsearch集成，保存原始请求/响应
- 🔔 **智能告警** - 邮件/钉钉/企业微信/Webhook
- 📍 **IP查询** - 地理位置查询功能
- ⚡ **高性能** - 并发检查（100 workers），资源高效
- 🎨 **响应式UI** - 现代化界面设计，60秒自动刷新
- 📈 **统计分析** - 正常运行时间、响应时间趋势

---

## 🚀 快速开始

### 安装运行

```bash
# 1. 编译
make build

# 2. 运行
go run cmd/server/main.go

# 3. 访问Web界面
open http://localhost:8080
```

---

## 📱 核心功能

### 监控管理
- ✅ **添加监控** - 完整的配置表单
  - 域名详情显示（域名、DNS解析的IP地址）
  - SSL证书监控（获取证书链信息，包括过期时间、颁发机构、组织等）
  - 支持完整URL输入（自动检测协议和端口）
  - DNS供应商选择或自定义IP绑定
  - 常用HTTP请求头预设
- ✅ **编辑监控** - 修改现有配置
- ✅ **删除监控** - 一键删除（自动清理关联数据）
- ✅ **实时状态** - 在线/离线/响应时间
- ✅ **正常运行时间** - 30天统计，可视化进度条
- ✅ **立即检查** - 创建后立即触发一次拨测

### 日志查询
- 🔍 **多条件搜索** - 目标/状态/时间范围
- 📄 **分页浏览** - 大量日志轻松查看
- 📋 **详细信息** - 完整请求/响应（请求包、响应包大小）
- ⏰ **时间范围** - 灵活的时间筛选
- 💾 **文件存储** - 无需ES也能使用（JSONL格式）

### IP查询
- 🌍 **地理位置** - 国家/地区/城市/ISP
- 📍 **坐标显示** - 精确经纬度

---

## ⚙️ 配置说明

### 基础配置 (config.yaml)

```yaml
server:
  http_port: 8080
  grpc_port: 9090
  host: 0.0.0.0

database:
  driver: sqlite
  dbname: monitor.db

monitor:
  check_interval: 60
  workers: 100

logger:
  level: info
  output: stdout

elasticsearch:
  enabled: false
  addresses:
    - http://localhost:9200
```

---

## 📊 监控类型

### HTTP/HTTPS
- ✅ 支持完整URL输入（自动检测协议和端口）
- ✅ 自定义HTTP方法（GET/POST/PUT/DELETE等）
- ✅ 常用HTTP请求头预设（User-Agent、Accept等）
- ✅ 自定义请求头和请求体
- ✅ 自定义Host头/IP绑定
- ✅ DNS供应商选择或自定义DNS服务器
- ✅ **SSL/TLS证书监控**（HTTPS类型）
  - 获取完整证书链（终端实体、中间、根证书）
  - 证书过期时间、颁发机构、组织、DNS名称
  - 证书序列号、指纹
- ✅ **DNS解析的IP地址**（真实IP，非域名）

### TCP/UDP
- ✅ 端口连通性检查
- ✅ 响应时间测量

### DNS
- ✅ 域名解析检查
- ✅ 自定义DNS服务器
- ✅ DNS记录保存

---

## 🏗️ 项目结构

```
monitor/
├── cmd/server/          # 程序入口
├── internal/            # 内部包
│   ├── alert/          # 告警引擎
│   ├── config/         # 配置管理
│   ├── database/       # 数据库
│   ├── elasticsearch/  # ES客户端
│   ├── logger/         # 日志（含文件日志）
│   ├── models/         # 数据模型
│   └── monitor/        # 监控服务
│       ├── https.go    # HTTPS+SSL证书检查器
│       ├── http.go     # HTTP检查器（含DNS解析）
│       ├── ssl.go      # SSL证书链获取
│       └── service.go  # 监控服务（100 workers）
├── api/server/         # HTTP服务器
├── web/                # Web界面
│   ├── static/
│   │   ├── css/style.css
│   │   └── js/app.js   # 前端逻辑
│   └── templates/index.html
├── pkg/ipgeo/          # IP查询
└── logs/               # 文件日志目录（自动创建）
```

---

## 🎯 技术栈

| 类别 | 技术 |
|------|------|
| 后端 | Go 1.24, Gin, GORM, Zap |
| 前端 | JavaScript, CSS3, Font Awesome |
| 数据库 | SQLite/MySQL/PostgreSQL |
| 日志 | Elasticsearch 8.x (可选) + 文件日志（JSONL）|
| 监控 | HTTP/HTTPS, TCP/UDP, DNS, SSL/TLS |

---

## 💡 使用示例

### 示例1: 监控HTTPS网站并检查SSL证书

通过Web界面添加：
```
名称: 测试
类型: HTTPS
地址: https://www.baidu.com  # 支持完整URL
端口: 443（自动检测）
SSL证书监控: ✅ 勾选
检查间隔: 60秒
启用监控: ✅
```

创建后会立即进行一次拨测，监控详情显示：
- 域名详情：www.baidu.com + DNS解析的IP
- SSL证书链：所有证书的详细信息

### 示例2: 监控API接口

```
名称: 用户API
类型: HTTPS
地址: api.example.com/user/info
HTTP方法: POST
请求体: {"action":"ping"}
HTTP请求头:
  - Authorization: Bearer your-token  # 从预设选择
  - Content-Type: application/json
DNS服务器: 8.8.8.8  # 自定义DNS
检查间隔: 30秒
```

### 示例3: 监控数据库端口

```
名称: MySQL主库
类型: TCP
地址: db.example.com
端口: 3306
检查间隔: 60秒
```

---

## 📚 详细文档

📖 **完整文档**: 请查看 [DOCUMENTATION.md](DOCUMENTATION.md) 获取：
- 完整API接口文档
- Web界面使用指南
- 系统配置详解
- 部署运维指南
- 故障排查指南

---

## 📄 许可证

Apache License 2.0

---

<div align="center">

**⬆ 返回顶部**

**Made with ❤️**

**项目完成日期**: 2025-01-11
**版本**: v0.1
**状态**: ✅ 生产就绪

**查看详细文档**: [DOCUMENTATION.md](DOCUMENTATION.md)

</div>
