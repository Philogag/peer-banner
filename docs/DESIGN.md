# qBittorrent Leecher Banner

一个Go语言编写的命令行Daemon工具，用于定时检测qBittorrent服务器中的吸血用户（Leecher），并生成DAT格式的屏蔽列表。

## 功能特性

- 支持多个qBittorrent服务器
- 灵活可配置的吸血判定规则
- 支持多种DAT输出格式
- 支持白名单管理
- 试运行模式（dry-run）
- 支持作为systemd服务运行

## 目录结构

```
qbittorrent-banner/
├── main.go                 # 程序入口
├── go.mod                  # Go模块文件
├── config.yaml             # 配置文件
├── internal/
│   ├── config/             # 配置加载
│   │   └── config.go
│   ├── api/                # qBittorrent API客户端
│   │   └── client.go
│   ├── models/             # 数据模型
│   │   └── models.go
│   ├── detector/           # 吸血检测引擎
│   │   └── detector.go
│   ├── rules/              # 判定规则实现
│   │   ├── rule.go         # 规则接口
│   │   └── filter.go       # 过滤条件定义
│   └── output/             # 输出处理器
│       └── dat_writer.go   # DAT文件生成
└── docs/
    └── DESIGN.md           # 本设计文档
```

## 配置文件

### 完整配置示例

```yaml
# 应用基础配置
app:
  interval: 30             # 检查间隔（分钟）
  log_level: info          # debug/info/warn/error
  dry_run: false           # 试运行模式，不写入文件
  state_file: bans.json    # 封禁状态文件路径

# qBittorrent 服务器配置
servers:
  - name: "My Server"
    url: "http://localhost:8080"
    username: "admin"
    password: "your_password"
    # 可添加多个服务器

# 白名单配置
whitelist:
  ips:
    - "192.168.1.1"
    - "10.0.0.0/8"

# 输出配置
output:
  dat_file: "/var/lib/qbittorrent/noLeech.dat"
  format: "peerbanana"     # peerbanana / plain

# 吸血判定规则配置
rules:
  # 规则1: 低分享率吸血用户（首次封禁24小时，3次后永封）
  - name: "low_share_leecher"
    enabled: true
    ban_duration: 24h      # 封禁时长（24小时）
    max_ban_count: 3       # 达到3次后永久封禁
    action: "ban"
    filter:
      - field: "downloaded"
        operator: ">="
        value: "1GB"
      - field: "uploaded"
        operator: "<"
        value: "50%"

  # 规则2: 进度达到100%但上传极低（封禁7天）
  - name: "completed_low_upload"
    enabled: true
    ban_duration: 168h     # 7天
    max_ban_count: 0        # 不启用自动永封
    action: "ban"
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

  # 规则3: 永久封禁（不设置 ban_duration 或设为 0）
  - name: "fake_client"
    enabled: true
    ban_duration: 0         # 永久封禁
    max_ban_count: 0
    action: "ban"
    filter:
      - field: "client"
        operator: "include"
        value: "FakeClient"

  # 规则4: 轻度警告（封禁1小时，多次后永封）
  - name: "minor_infractor"
    enabled: true
    ban_duration: 1h        # 1小时
    max_ban_count: 5        # 5次后永封
    action: "ban"
    filter:
      - field: "progress"
        operator: "<"
        value: "10"
      - field: "uploaded"
        operator: "<"
        value: "100MB"
```

---

# 吸血判定规则配置

吸血用户判定采用 **AND** 组合规则，即一个用户必须**同时满足**所有指定条件的阈值，才会被判定为吸血用户。

## 规则配置结构

```yaml
rules:
  # 规则名称，用于标识
  - name: "rule_name"
    enabled: true
    # 触发动作：ban（加入黑名单）、warn（仅记录）
    action: "ban"
    # 过滤条件（所有条件需同时满足）
    filter:
      - field: "progress"     # 过滤字段
        operator: "<"         # 操作符
        value: "10"           # 值（自动识别单位）
```

## 过滤条件结构 (Filter)

每个过滤条件包含以下字段：

| 字段 | 必填 | 说明 |
|------|------|------|
| `field` | 是 | 过滤指标字段名 |
| `operator` | 是 | 操作符：`<`, `>`, `<=`, `>=`, `include`, `exclude` |
| `value` | 是 | 值，自动识别百分比或字节单位 |

### 支持的字段 (Field)

| 字段 | 说明 | 值示例 |
|------|------|--------|
| `progress` | 下载进度 (0-100) | `"50"`, `"99.5"` |
| `uploaded` | 已上传字节量/百分比 | `"50%"`, `"1GB"`, `"512KB"` |
| `downloaded` | 已下载字节量/百分比 | `"1GB"`, `"50%"` |
| `relevance` | 文件关联度 (0-1) | `"0.3"`, `"0.5"` |
| `active_time` | 活动时间 | `"24h"`, `"7d"`, `"1h30m"` |
| `flag` | 客户端标志 | `"encrypted"`, `"i2p"` |

### 支持的操作符 (Operator)

| 操作符 | 说明 | 适用类型 |
|--------|------|----------|
| `<` | 小于 | 数值、百分比、字节 |
| `>` | 大于 | 数值、百分比、字节 |
| `<=` | 小于等于 | 数值、百分比、字节 |
| `>=` | 大于等于 | 数值、百分比、字节 |
| `include` | 包含 | 字符串、列表 |
| `exclude` | 不包含 | 字符串、列表 |

### 值格式 (Value)

#### 百分比
直接使用数字加 `%` 后缀：
```yaml
value: "50%"      # 50%
value: "0.5%"     # 0.5%
```

#### 字节单位
支持 `B`, `KB`, `MB`, `GB`, `TB`：
```yaml
value: "1GB"      # 1 GB
value: "512KB"    # 512 KB
value: "100MB"    # 100 MB
value: "2TB"      # 2 TB
```

#### 时间单位
支持 `s`(秒), `m`(分钟), `h`(小时), `d`(天)：
```yaml
value: "24h"      # 24 小时
value: "7d"       # 7 天
value: "1h30m"    # 1 小时 30 分钟
```

#### 数值
直接使用数字（用于 progress, relevance 等）：
```yaml
value: "99"       # 进度 99
value: "0.3"      # 关联度 0.3
```

---

## 组合规则示例

以下配置演示如何使用多个指标的 **AND** 组合：

### 示例 1：检测下载超过 50% 但上传不足 5% 的用户

```yaml
rules:
  - name: "download_high_upload_low"
    enabled: true
    action: "ban"
    filter:
      - field: "downloaded"
        operator: ">="
        value: "50%"
      - field: "uploaded"
        operator: "<"
        value: "5%"
```

### 示例 2：检测长时间挂机但贡献极低的用户

```yaml
rules:
  - name: "long_active_low_share"
    enabled: true
    action: "ban"
    filter:
      - field: "active_time"
        operator: ">="
        value: "24h"           # 超过 24 小时
      - field: "progress"
        operator: ">="
        value: "80"            # 进度超过 80%
      - field: "uploaded"
        operator: "<"
        value: "10%"           # 但上传不足 10%
```

### 示例 3：检测已下载大量资源但几乎不上传的用户

```yaml
rules:
  - name: "heavy_download_no_upload"
    enabled: true
    action: "ban"
    filter:
      - field: "downloaded"
        operator: ">="
        value: "5GB"           # 下载超过 5GB
      - field: "uploaded"
        operator: "<"
        value: "512KB"         # 上传不足 512KB
```

### 示例 4：检测使用加密连接但进度长期停滞的用户

```yaml
rules:
  - name: "encrypted_stalled"
    enabled: false
    action: "ban"
    filter:
      - field: "flag"
        operator: "include"
        value: "encrypted"     # 使用加密连接
      - field: "progress"
        operator: "<"
        value: "1"             # 进度小于 1%
      - field: "active_time"
        operator: ">="
        value: "1h"            # 连接超过 1 小时
```

### 示例 5：检测关联度低且上传贡献极差的用户

```yaml
rules:
  - name: "low_relevance_low_upload"
    enabled: true
    action: "ban"
    filter:
      - field: "relevance"
        operator: "<"
        value: "0.2"           # 关联度低于 20%
      - field: "uploaded"
        operator: "<"
        value: "10%"           # 上传低于 10%
      - field: "downloaded"
        operator: ">="
        value: "50%"           # 下载超过 50%
```

### 示例 6：排除特定标志的用户

```yaml
rules:
  - name: "not_i2p_users"
    enabled: true
    action: "ban"
    filter:
      - field: "flag"
        operator: "exclude"
        value: "i2p"           # 排除 I2P 用户
      - field: "progress"
        operator: "<"
        value: "10"
      - field: "active_time"
        operator: ">="
        value: "2h"
```

---

## 配置项说明

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `app.interval` | int | 检查间隔，单位分钟 |
| `app.log_level` | string | 日志级别 (debug/info/warn/error) |
| `app.dry_run` | bool | 试运行模式，不写入文件 |
| `servers[].name` | string | 服务器名称 |
| `servers[].url` | string | qBittorrent Web API 地址 |
| `servers[].username` | string | 用户名 |
| `servers[].password` | string | 密码 |
| `whitelist.ips` | []string | 白名单 IP/网段 |
| `output.dat_file` | string | 输出 DAT 文件路径 |
| `output.format` | string | 输出格式 (peerbanana/plain) |
| `rules[].name` | string | 规则标识符 |
| `rules[].enabled` | bool | 是否启用 |
| `rules[].action` | string | 触发动作 (ban/warn) |
| `rules[].filter[].field` | string | 过滤字段名 |
| `rules[].filter[].operator` | string | 操作符 |
| `rules[].filter[].value` | string | 值（自动识别单位） |

---

## 判定规则设计

### 规则接口

```go
type Filter interface {
    // 检查 peer 是否满足该条件
    Match(peer *Peer, torrent *Torrent) bool
}

// Rule 定义吸血判定规则
type Rule struct {
    Name     string        `yaml:"name"`
    Enabled  bool          `yaml:"enabled"`
    Action   string        `yaml:"action"` // ban, warn
    Filters  []FilterConfig `yaml:"filter"`
}

// FilterConfig 配置化的过滤条件
type FilterConfig struct {
    Field    string `yaml:"field"`
    Operator string `yaml:"operator"` // <, >, <=, >=, include, exclude
    Value    string `yaml:"value"`
}
```

### 检测引擎工作流程

```
1. 获取所有种子列表
2. 获取每个种子的 peer 信息
3. 遍历每个 peer，应用所有规则
4. 对于每个规则：
   a. 检查 peer 是否满足该规则的所有 filter（AND 组合）
   b. 如果满足，将该 peer 标记为吸血用户
5. 收集所有被标记的 IP
6. 生成 DAT 文件
```

---

## 输出格式

### PeerBanana格式

```
# PeerBanana DAT File
# Generated by qBittorrent Leecher Banner
# Date: 2024-01-01 00:00:00

# Banned IPs
123.45.67.89
98.76.54.32
```

### Plain格式

```
123.45.67.89
98.76.54.32
```

---

## 使用方法

### 编译

```bash
go build -o qbittorrent-banner .
```

### 运行

```bash
# 使用默认配置文件
./qbittorrent-banner

# 指定配置文件
./qbittorrent-banner -config=/path/to/config.yaml

# 仅运行一次
./qbittorrent-banner -once

# 试运行模式
./qbittorrent-banner -dry-run

# 显示版本
./qbittorrent-banner -version
```

### 安装为Systemd服务

```ini
# /etc/systemd/system/qbittorrent-banner.service
[Unit]
Description=qBittorrent Leecher Banner
After=network.target

[Service]
Type=simple
User=qbittorrent
Group=qbittorrent
ExecStart=/usr/local/bin/qbittorrent-banner -config=/etc/qbittorrent-banner.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable qbittorrent-banner
sudo systemctl start qbittorrent-banner
```

---

## API调用

qBittorrent Web API:

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v2/auth/login` | POST | 用户认证 |
| `/api/v2/torrents/info` | GET | 获取种子列表 |
| `/api/v2/torrents/properties` | GET | 获取种子详情 |
| `/api/v2/torrents/trackers` | GET | 获取种子 trackers |
| `/api/v2/sync/maindata` | GET | 同步数据 |
| `/api/v2/sync/torrentPeers` | GET | 获取 peers 状态 |

---

## 扩展开发

### 添加新的过滤指标

1. 在 `FilterConfig` 结构体中添加新字段
2. 实现 `Filter` 接口
3. 在检测引擎中注册新指标

```go
// 示例：添加新的 Filter
type MyFilter struct {
    Threshold float64 `yaml:"threshold"`
    Operator  string  `yaml:"operator"`
}

func (f *MyFilter) Match(peer *Peer, torrent *Torrent) bool {
    // 根据 operator 进行比较
    switch f.Operator {
    case "<":
        return peer.SomeValue < f.Threshold
    case ">":
        return peer.SomeValue > f.Threshold
    // ... 其他操作符
    }
    return false
}
```

---

---

## Docker 部署

### 构建镜像

```bash
# 构建镜像
docker build -t qbittorrent-banner .

# 或者使用多阶段构建
docker build -t qbittorrent-banner:latest --target release .
```

### 配置文件

创建 `config.yaml` 配置文件：

```yaml
app:
  interval: 30
  log_level: info
  dry_run: false

servers:
  - name: "My Server"
    url: "http://host.docker.internal:8080"  # 使用 host.docker.internal 访问宿主机
    username: "admin"
    password: "password123"

output:
  dat_file: "/data/leechers.dat"
  format: "peerbanana"

rules:
  - name: "low_share"
    enabled: true
    action: "ban"
    filter:
      - field: "downloaded"
        operator: ">="
        value: "1GB"
      - field: "uploaded"
        operator: "<"
        value: "50%"
```

### 运行容器

```bash
# 基本运行
docker run -d \
  --name qbittorrent-banner \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/data:/data \
  qbittorrent-banner

# 查看日志
docker logs -f qbittorrent-banner
```

### Docker Compose 部署

```yaml
# docker-compose.yml
version: "3.8"

services:
  qbittorrent-banner:
    image: qbittorrent-banner:latest
    container_name: qbittorrent-banner
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
```

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart
```

### ARM/Raspberry Pi 部署

```bash
# 构建 ARM64 镜像
docker build -t qbittorrent-banner:latest --platform linux/arm64 .

# 或者使用 Buildx
docker buildx build -t qbittorrent-banner:latest --platform amd64,arm64 .
```

### 定时任务（可选）

如果不想持续运行，可以使用 Docker 的定时任务：

```yaml
# docker-compose.yml
version: "3.8"

services:
  qbittorrent-banner:
    image: qbittorrent-banner:latest
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/data
    entrypoint: ["/bin/sh", "-c"]
    command:
      - "while true; do /app/qbittorrent-banner -config=/app/config.yaml; sleep 1800; done"
```

或者使用宿主机的 cron：

```bashetc/cron.d
# //qbittorrent-banner
0 */1 * * * docker run --rm -v /path/to/config.yaml:/app/config.yaml -v /path/to/data:/data qbittorrent-banner -config=/app/config.yaml -once >> /var/log/qbittorrent-banner.log 2>&1
```

---

## 许可证

MIT License

---

## 封禁时长功能 (Ban Duration)

### 功能概述

支持配置 IP 封禁时长，到期后自动解除封禁，无需手动管理。

### 配置示例

```yaml
rules:
  # 规则1: 临时封禁 24 小时
  - name: "low_share_leecher"
    enabled: true
    ban_duration: 24h      # 封禁 24 小时
    action: "ban"
    filter:
      - field: "downloaded"
        operator: ">="
        value: "1GB"
      - field: "uploaded"
        operator: "<"
        value: "50%"

  # 规则2: 永久封禁
  - name: "permanent_ban"
    enabled: true
    ban_duration: 0        # 0 或留空表示永久封禁
    action: "ban"
    filter:
      - field: "client"
        operator: "include"
        value: "FakeClient"

  # 规则3: 封禁 7 天
  - name: "zero_uploader"
    enabled: true
    ban_duration: 168h      # 7 天 = 168 小时
    action: "ban"
    filter:
      - field: "uploaded"
        operator: "=="
        value: "0"
```

### 时长格式

| 格式 | 示例 | 说明 |
|------|------|------|
| 小时 | `24h` | 24 小时 |
| 分钟 | `30m` | 30 分钟 |
| 天 | `7d` | 7 天 |
| 周 | `2w` | 2 周 |
| 永久 | `0` 或留空 | 永不解封 |

### 状态文件

系统会自动生成 `bans.json` 文件记录封禁状态：

```json
{
  "version": 1,
  "last_updated": "2026-02-05T10:00:00Z",
  "bans": {
    "192.168.1.100": {
      "ip": "192.168.1.100",
      "reason": "Matched rule: low_share_leecher",
      "rule_name": "low_share_leecher",
      "banned_at": "2026-02-05T08:00:00Z",
      "expires_at": "2026-02-06T08:00:00Z"
    },
    "192.168.1.101": {
      "ip": "192.168.1.101",
      "reason": "Matched rule: permanent_ban",
      "rule_name": "permanent_ban",
      "banned_at": "2026-02-05T08:00:00Z",
      "expires_at": "0001-01-01T00:00:00Z"
    }
  }
}
```

### 工作流程

```
1. 启动时加载 bans.json，读取所有封禁记录
2. 检测时跳过已过期的 IP（仅保留未过期的到新结果）
3. 新封禁的 IP 根据规则配置写入 expires_at
4. 每次检测后更新 bans.json 文件
```

### 新增配置项

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `app.state_file` | string | `bans.json` | 封禁状态文件路径 |
| `rules[].ban_duration` | string | `0` (永久) | 封禁时长 |
| `rules[].max_ban_count` | int | `0` | 达到此封禁次数后永封，0表示不禁用 |

---

## 自动永封功能 (Escalate to Permanent Ban)

### 功能概述

当某个 IP 被临时封禁达到指定次数后，自动升级为永久封禁。适用于屡教不改的吸血用户。

### 配置示例

```yaml
rules:
  # 规则1: 首次封禁24小时，3次后永封
  - name: "low_share_leecher"
    enabled: true
    ban_duration: 24h        # 首次封禁 24 小时
    max_ban_count: 3         # 3 次后永久封禁
    action: "ban"
    filter:
      - field: "downloaded"
        operator: ">="
        value: "1GB"
      - field: "uploaded"
        operator: "<"
        value: "50%

  # 规则2: 首次封禁1小时，5次后永封
  - name: "temporary_warner"
    enabled: true
    ban_duration: 1h         # 首次封禁 1 小时
    max_ban_count: 5         # 5 次后永久封禁
    action: "ban"
    filter:
      - field: "progress"
        operator: "<"
        value: "10"

  # 规则3: 不使用自动永封功能
  - name: "always_temporary"
    enabled: true
    ban_duration: 24h
    max_ban_count: 0         # 0 表示禁用此功能，永远临时封禁
    action: "ban"
    filter:
      - field: "uploaded"
        operator: "=="
        value: "0"
```

### 行为说明

| 场景 | 行为 |
|------|------|
| IP 首次被封禁 | 按 `ban_duration` 封禁 |
| 第 2 次被封禁 | 再次按 `ban_duration` 封禁 |
| 达到 `max_ban_count` 次 | 永久封禁（`expires_at` 为空） |
| `max_ban_count` 为 0 | 每次都是临时封禁，永不永封 |

### 状态文件扩展

```json
{
  "version": 2,
  "last_updated": "2026-02-05T10:00:00Z",
  "bans": {
    "192.168.1.100": {
      "ip": "192.168.1.100",
      "reason": "Matched rule: low_share_leecher",
      "rule_name": "low_share_leecher",
      "banned_at": "2026-02-05T08:00:00Z",
      "expires_at": "2026-02-05T09:00:00Z",
      "ban_count": 1,
      "is_permanent": false
    },
    "192.168.1.101": {
      "ip": "192.168.1.101",
      "reason": "Escalated to permanent ban after 3 violations",
      "rule_name": "low_share_leecher",
      "banned_at": "2026-02-04T08:00:00Z",
      "expires_at": "0001-01-01T00:00:00Z",
      "ban_count": 3,
      "is_permanent": true
    }
  }
}
```

### 新增字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `ban_count` | int | 该 IP 被封禁的累计次数 |
| `is_permanent` | bool | 是否已升级为永久封禁 |
| `version` | int | 状态文件版本，升级后为 2 |

### 升级路径

```
ban_count: 1 ──封禁 24h──▶ ban_count: 2 ──封禁 24h──▶ ban_count: 3 (永封)
                                                                 │
                                                                 ▼
                                                      is_permanent: true
                                                      expires_at: null
```

### 完整配置项

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `app.state_file` | string | `bans.json` | 封禁状态文件路径 |
| `rules[].ban_duration` | string | `0` (永久) | 封禁时长 |
| `rules[].max_ban_count` | int | `0` | 达到此次数后永封，0表示禁用 |
