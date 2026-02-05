# Peer Banner - qBittorrent 吸血用户检测工具

一个用 Go 语言编写的命令行工具，用于定时检测 qBittorrent 服务器中的吸血用户（Leecher），并生成 DAT 格式的屏蔽列表。

## 功能特性

- 支持多个 qBittorrent 服务器
- 灵活可配置的吸血判定规则（AND 组合）
- 支持多种 DAT 输出格式
- 支持白名单管理
- 试运行模式（dry-run）
- 封禁时长管理
- 自动升级为永久封禁
- 支持 systemd 服务运行

## 安装

### 从源码编译

```bash
git clone https://github.com/philogag/peer-banner.git
cd peer-banner
go build -o peer-banner .
```

### 二进制下载

从 [Releases](https://github.com/philogag/peer-banner/releases) 下载预编译的二进制文件。

## 使用方法

```bash
# 使用默认配置文件
./peer-banner

# 指定配置文件
./peer-banner -config=/path/to/config.yaml

# 仅运行一次检测
./peer-banner -once

# 试运行模式（不写入文件）
./peer-banner -dry-run

# 显示版本信息
./peer-banner -version
```

## 配置文件

### 完整配置示例

```yaml
# 应用基础配置
app:
  interval: 30              # 检查间隔（分钟）
  log_level: info           # debug/info/warn/error
  dry_run: false            # 试运行模式
  state_file: bans.json     # 封禁状态文件路径

# qBittorrent 服务器配置
servers:
  - name: "Main Server"
    url: "http://localhost:8080"
    username: "admin"
    password: "your_password"
  # 可以添加更多服务器
  # - name: "Backup Server"
  #   url: "http://192.168.1.100:8080"
  #   username: "admin"
  #   password: "password"

# 白名单配置（这些IP不会被ban）
whitelist:
  ips:
    - "127.0.0.1"
    - "192.168.1.0/24"
    - "10.0.0.0/8"

# 输出配置
output:
  dat_file: "/data/leechers.dat"
  format: "peerbanana"     # peerbanana / plain

# 吸血判定规则配置
# 使用 AND 组合：用户必须同时满足所有 filter 条件才会被判定为吸血用户
rules:
  # 规则 1: 下载超过 1GB 但上传低于 50%
  - name: "low_share_leecher"
    enabled: true
    action: "ban"
    ban_duration: "24h"     # 封禁时长（0 表示永久）
    max_ban_count: 3        # 达到此次数后永久封禁
    filter:
      - field: "downloaded"
        operator: ">="
        value: "1GB"
      - field: "uploaded"
        operator: "<"
        value: "50%"

  # 规则 2: 进度达到 99%+ 但上传不足 20%
  - name: "completed_low_upload"
    enabled: true
    action: "ban"
    ban_duration: "168h"    # 7天
    filter:
      - field: "progress"
        operator: ">="
        value: "99"
      - field: "uploaded"
        operator: "<"
        value: "20%"
      - field: "relevance"
        operator: "<"
        value: "0.3"

  # 规则 3: 活动超过 24 小时但上传不足 1%
  - name: "stalled_seeder"
    enabled: true
    action: "ban"
    filter:
      - field: "active_time"
        operator: ">="
        value: "24h"
      - field: "uploaded"
        operator: "<"
        value: "1%"

  # 规则 4: 使用加密协议且进度长期停滞
  - name: "encrypted_stalled"
    enabled: false
    action: "ban"
    filter:
      - field: "flag"
        operator: "include"
        value: "encrypted"
      - field: "progress"
        operator: "<"
        value: "5"
      - field: "active_time"
        operator: ">="
        value: "1h"
```

## 配置项说明

### App 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `interval` | int | 30 | 检查间隔（分钟） |
| `log_level` | string | info | 日志级别 (debug/info/warn/error) |
| `dry_run` | bool | false | 试运行模式 |
| `state_file` | string | bans.json | 封禁状态文件路径 |

### Server 配置

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `name` | string | 服务器名称 |
| `url` | string | qBittorrent Web API 地址 |
| `username` | string | 用户名 |
| `password` | string | 密码 |

### Output 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `dat_file` | string | - | 输出 DAT 文件路径 |
| `format` | string | peerbanana | 输出格式 (peerbanana/plain) |

### Rule 配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `name` | string | - | 规则名称 |
| `enabled` | bool | true | 是否启用 |
| `action` | string | ban | 触发动作 (ban/warn) |
| `ban_duration` | string | 0 | 封禁时长 (0 表示永久) |
| `max_ban_count` | int | 0 | 达到此次数后永封 |
| `filter` | []Filter | - | 过滤条件列表 |

### Filter 配置

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `field` | string | 过滤字段 |
| `operator` | string | 操作符 |
| `value` | string | 值 |

### 支持的过滤字段

| 字段 | 说明 | 值示例 |
|------|------|--------|
| `progress` | 下载进度 (0-100) | `50`, `99.5` |
| `uploaded` | 已上传量 | `50%`, `1GB`, `512KB` |
| `downloaded` | 已下载量 | `1GB`, `50%` |
| `relevance` | 文件关联度 (0-1) | `0.3`, `0.5` |
| `active_time` | 活动时间（秒） | `86400`, `24h` |
| `flag` | 客户端标志 | `encrypted`, `i2p` |

### 支持的操作符

| 操作符 | 说明 |
|--------|------|
| `<` | 小于 |
| `>` | 大于 |
| `<=` | 小于等于 |
| `>=` | 大于等于 |
| `include` | 包含 |
| `exclude` | 不包含 |

### 值格式

- **百分比**: `50%`, `0.5%`
- **字节**: `1GB`, `512KB`, `100MB`, `2TB`
- **时间**: `24h` (小时), `7d` (天), `1h30m` (复合)
- **数值**: `99`, `0.3`

## 安装为 Systemd 服务

创建 `/etc/systemd/system/peer-banner.service`:

```ini
[Unit]
Description=Peer Banner - qBittorrent Leecher Detection
After=network.target

[Service]
Type=simple
User=qbittorrent
Group=qbittorrent
ExecStart=/usr/local/bin/peer-banner -config=/etc/peer-banner.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable peer-banner
sudo systemctl start peer-banner
```

## Docker 部署

### 构建镜像

```bash
docker build -t peer-banner .
```

### 运行容器

```bash
docker run -d \
  --name peer-banner \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/data:/data \
  peer-banner
```

### Docker Compose

```yaml
version: "3.8"

services:
  peer-banner:
    image: peer-banner:latest
    container_name: peer-banner
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
```

## 项目结构

```
peer-banner/
├── main.go                 # 程序入口
├── go.mod                  # Go 模块文件
├── config.example.yaml     # 配置文件示例
├── internal/
│   ├── api/               # qBittorrent API 客户端
│   ├── ban/               # 封禁状态管理
│   ├── config/            # 配置加载
│   ├── detector/          # 吸血检测引擎
│   ├── models/            # 数据模型
│   ├── output/            # DAT 文件输出
│   └── rules/             # 判定规则实现
└── docs/
    └── DESIGN.md          # 设计文档
```

## 输出格式

### PeerBanana 格式

```
# PeerBanana DAT File
# Generated by Peer Banner
# Date: 2024-01-01 00:00:00

# Banned IPs
123.45.67.89
98.76.54.32
```

### Plain 格式

```
123.45.67.89
98.76.54.32
```

## 许可证

MIT License
