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
│   │   └── criteria.go     # 条件定义
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
  interval: 30          # 检查间隔（分钟）
  log_level: info      # debug/info/warn/error
  dry_run: false        # 试运行模式，不写入文件

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
  format: "peerbanana"  # peerbanana / plain

# 吸血判定规则配置
rules:
  # 规则1: 低分享率吸血用户
  - name: "low_share_leecher"
    enabled: true
    action: "ban"
    criteria:
      downloaded:
        mode: "absolute"
        min: "1GB"
      uploaded:
        mode: "percent"
        max: "50%"

  # 规则2: 进度达到100%但上传极低
  - name: "completed_low_upload"
    enabled: true
    action: "ban"
    criteria:
      progress:
        min: 99
      uploaded:
        mode: "percent"
        max: "20%"
      relevance:
        max: 0.3

  # 规则3: 长时间挂机不活跃上传
  - name: "stalled_seeder"
    enabled: true
    action: "ban"
    criteria:
      active_time:
        min: "24h"
      uploaded:
        mode: "percent"
        max: "1%"
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
    # 判定条件（所有条件需同时满足）
    criteria:
      # 以下任一条件满足即触发
```

## 支持的过滤指标 (Criteria)

每个规则的 `criteria` 字段可以组合使用以下指标：

### 1. 特定 Flag 存在 (flag)

检测 peer 是否包含指定的 qBittorrent flag。

```yaml
criteria:
  flag:
    - "encrypted"      # BT 加密连接
    - "i2p"            # I2P 网络
    - "pex"            # PEX 已启用
    - "dht"            # DHT 已启用
    - "lt_pex"         # LPEX 已启用
    - "ut_pex"         # uTPEX 已启用
    - "ssl"            # SSL 加密
```

**匹配逻辑**：peer 包含列表中任一 flag 即满足该条件。

### 2. 客户端进度 (progress)

检测 peer 的下载进度百分比。

```yaml
criteria:
  progress:
    min: 0.0           # 最小进度 (0-100%)
    max: 10.0          # 最大进度
```

**匹配逻辑**：`min <= peer.progress <= max`

### 3. 客户端已上传 (uploaded)

检测 peer 已上传的字节量，支持**百分比阈值**或**绝对值**。

```yaml
# 百分比模式（相对于种子总大小）
criteria:
  uploaded:
    mode: "percent"    # 模式：percent 或 absolute
    min_percent: 0      # 最小上传百分比
    max_percent: 5      # 最大上传百分比

# 绝对值模式
criteria:
  uploaded:
    mode: "absolute"
    min: "0"           # 最小上传量
    max: "50MB"         # 最大上传量
```

**支持的单位**：B, KB, MB, GB, TB

**百分比计算**：`uploaded / torrent_size * 100`

### 4. 客户端已下载 (downloaded)

检测 peer 已下载的字节量，支持**百分比阈值**或**绝对值**。

```yaml
# 百分比模式
criteria:
  downloaded:
    mode: "percent"
    min_percent: 50     # 最小下载百分比
    max_percent: 100    # 最大下载百分比

# 绝对值模式
criteria:
  downloaded:
    mode: "absolute"
    min: "1GB"          # 最小下载量
    max: "1TB"          # 最大下载量
```

### 5. 文件关联度 (relevance)

检测 peer 对种子的文件关联度/分享率 (0.0 - 1.0)。

```yaml
criteria:
  relevance:
    min: 0.0           # 最小关联度
    max: 0.3           # 最大关联度
```

**说明**：关联度表示 peer 拥有的稀缺块比例，越低说明该 peer 拥有的块越普遍。

### 6. 种子活动时间 (active_time)

检测 peer 在种子上的活动时长。

```yaml
criteria:
  active_time:
    min: "1h"          # 最小活动时间
    max: "7d"          # 最大活动时间
```

**支持单位**：`s`(秒), `m`(分钟), `h`(小时), `d`(天)

---

## 组合规则示例

以下配置演示如何使用多个指标的 **AND** 组合：

### 示例 1：检测下载超过 50% 但上传不足 5% 的用户

```yaml
rules:
  - name: "download_high_upload_low"
    enabled: true
    action: "ban"
    criteria:
      downloaded:
        mode: "percent"
        min_percent: 50
      uploaded:
        mode: "percent"
        max_percent: 5
```

### 示例 2：检测长时间挂机但贡献极低的用户

```yaml
rules:
  - name: "long_active_low_share"
    enabled: true
    action: "ban"
    criteria:
      active_time:
        min: "24h"           # 超过 24 小时
      progress:
        min: 80              # 进度超过 80%
      uploaded:
        mode: "percent"
        max_percent: 10     # 但上传不足 10%
```

### 示例 3：检测已下载大量资源但几乎不上传的用户

```yaml
rules:
  - name: "heavy_download_no_upload"
    enabled: true
    action: "ban"
    criteria:
      downloaded:
        mode: "absolute"
        min: "5GB"           # 下载超过 5GB
      uploaded:
        mode: "absolute"
        max: "512KB"         # 上传不足 512KB
```

### 示例 4：检测使用加密连接但进度长期停滞的用户

```yaml
rules:
  - name: "encrypted_stalled"
    enabled: false
    action: "ban"
    criteria:
      flag:
        - "encrypted"
      progress:
        max: 1               # 进度小于 1%
      active_time:
        min: "1h"            # 连接超过 1 小时
```

### 示例 5：检测关联度低且上传贡献极差的用户

```yaml
rules:
  - name: "low_relevance_low_upload"
    enabled: true
    action: "ban"
    criteria:
      relevance:
        max: 0.2             # 关联度低于 20%
      uploaded:
        mode: "percent"
        max_percent: 10     # 上传低于 10%
      downloaded:
        mode: "percent"
        min_percent: 50    # 下载超过 50%
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
| `rules[].criteria.flag` | []string | 需要检测的 flag 列表 |
| `rules[].criteria.progress.min/max` | float | 进度百分比范围 (0-100) |
| `rules[].criteria.uploaded.mode` | string | 上传量模式 (percent/absolute) |
| `rules[].criteria.uploaded.min/max_percent` | float | 上传百分比范围 |
| `rules[].criteria.uploaded.min/max` | string | 上传字节数范围 (带单位) |
| `rules[].criteria.downloaded.mode` | string | 下载量模式 (percent/absolute) |
| `rules[].criteria.downloaded.min/max_percent` | float | 下载百分比范围 |
| `rules[].criteria.downloaded.min/max` | string | 下载字节数范围 (带单位) |
| `rules[].criteria.relevance.min/max` | float | 文件关联度范围 (0-1) |
| `rules[].criteria.active_time.min/max` | string | 活动时长范围 (带单位) |

---

## 判定规则设计

### 规则接口

```go
type Criteria interface {
    // 检查 peer 是否满足该条件
    Match(peer *Peer, torrent *Torrent) bool
}

// Rule 定义吸血判定规则
type Rule struct {
    Name      string            `yaml:"name"`
    Enabled   bool              `yaml:"enabled"`
    Action    string            `yaml:"action"` // ban, warn
    Criteria  []CriteriaConfig   `yaml:"criteria"`
}

// CriteriaConfig 配置化的条件
type CriteriaConfig struct {
    Flag        []string            `yaml:"flag,omitempty"`
    Progress    *ProgressCriteria   `yaml:"progress,omitempty"`
    Uploaded    *BytesCriteria      `yaml:"uploaded,omitempty"`
    Downloaded  *BytesCriteria      `yaml:"downloaded,omitempty"`
    Relevance   *RangeCriteria      `yaml:"relevance,omitempty"`
    ActiveTime  *TimeCriteria       `yaml:"active_time,omitempty"`
}
```

### 检测引擎工作流程

```
1. 获取所有种子列表
2. 获取每个种子的 peer 信息
3. 遍历每个 peer，应用所有规则
4. 对于每个规则：
   a. 检查 peer 是否满足该规则的所有 criteria（AND 组合）
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

1. 在 `CriteriaConfig` 结构体中添加新字段
2. 实现 `Criteria` 接口
3. 在检测引擎中注册新指标

```go
// 示例：添加新的 Criteria
type MyCriteria struct {
    Threshold float64 `yaml:"threshold"`
}

func (c *MyCriteria) Match(peer *Peer, torrent *Torrent) bool {
    return peer.SomeValue < c.Threshold
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
    criteria:
      downloaded:
        mode: "absolute"
        min: "1GB"
      uploaded:
        mode: "percent"
        max: "50%"
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
docker buildx build -t qbittorrent-banner:latest --platform linux/amd64,linux/arm64 .
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
