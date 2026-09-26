# 上海电信 IPTV 服务 API 与播放器配置指南

本文档详细介绍本服务提供的所有对外播放接口、管理面板 API、回看与时移技术规范，以及主流播放器（TiviMate、Televizo、APTV、DIYP 等）的实战配置。

---

## 目录

1. [系统概述与播放模式](#1-系统概述与播放模式)
2. [公共播放与订阅 API](#2-公共播放与订阅-api)
   - [2.1 播放列表导出 (`/api/playlist` 或 `/api/m3u8`)](#21-播放列表导出-apiplaylist-或-apim3u8)
   - [2.2 统一流媒体与回看重定向 (`/api/play`)](#22-统一流媒体与回看重定向-apiplay)
   - [2.3 XMLTV 电子节目单 (`/api/epg`)](#23-xmltv-电子节目单-apiepg)
   - [2.4 内嵌专用网页播放器 (`/player`)](#24-内嵌专用网页播放器-player)
   - [2.5 兼容性与根路径接口](#25-兼容性与根路径接口)
3. [管理面板 API (`/api/panel/...`)](#3-管理面板-api-apipanel)
   - [3.1 鉴权机制](#31-鉴权机制)
   - [3.2 系统状态与设置](#32-系统状态与设置)
   - [3.3 频道管理与探测](#33-频道管理与探测)
   - [3.4 频道模糊匹配与关键词映射](#34-频道模糊匹配与关键词映射)
   - [3.5 组播扫描与批量探测](#35-组播扫描与批量探测)
   - [3.6 台标管理与本地缓存](#36-台标管理与本地缓存)
   - [3.7 EPG 节目管理](#37-epg-节目管理)
4. [主流播放器实战配置指南](#4-主流播放器实战配置指南)
   - [4.1 TiviMate 配置 (推荐)](#41-tivimate-配置-推荐)
   - [4.2 Televizo 配置 (推荐)](#42-televizo-配置-推荐)
   - [4.3 APTV (Apple TV / iOS / macOS)](#43-aptv-apple-tv--ios--macos)
   - [4.4 DIYP / 影音壳子 / 影视仓](#44-diyp--影音壳子--影视仓)
   - [4.5 PotPlayer / VLC](#45-potplayer--vlc)
5. [回看与时移技术细节规范](#5-回看与时移技术细节规范)
6. [常见问题排查 (FAQ)](#6-常见问题排查-faq)

---

## 1. 系统概述与播放模式

本服务支持三种不同的电视播放模式，满足不同网络环境与硬件终端需求：

1. **组播直播 (Multicast - 默认)**
   - 地址格式：`http://<udpxy-address>/rtp/239.45.x.x:5140?fcc=...`
   - 特点：内网组播转单播（通过路由器 udpxy 或 msr），换台速度快，无公网流量损耗。
   - 依赖：局域网需有支持 IGMP Proxy / Snooping 的路由器及已配置的 udpxy 转发服务。
2. **RTSP 单播与回看 (Unicast - 经典)**
   - 直播地址：`rtsp://10.x.x.x:554/live/...`（自动通过转发服务包装为 HTTP 流）
   - 回看地址：通过 `/api/play?id=...&utc=...&lutc=...` 自动生成带 CST 时间参数的 `playseek=YYYYMMDDHHMMSS-YYYYMMDDHHMMSS` 重定向。
   - 特点：支持快进、快退、暂停、7 天完整回看与直播时移。
3. **运营商 HTTP 单播 (HTTP - 现代)**
   - 地址格式：通过 `/api/play?id=...&mode=http` 动态请求运营商门户接口，获取带时效性数字签名的真实播放流。
   - 特点：基于 HTTP 协议传输，穿透性好。

---

## 2. 公共播放与订阅 API

所有公共播放与订阅接口无需面板管理员 Token，专供各类电视盒子、手机或桌面播放器调用。

### 2.1 播放列表导出 (`/api/playlist` 或 `/api/m3u8`)

导出当前已启用的电视频道列表，支持 M3U、TXT 和 JSON 格式。

- **请求方法**：`GET`, `HEAD`
- **请求 URL**：`http://<server-ip>:8888/api/playlist` 或 `http://<server-ip>:8888/api/m3u8`
- **查询参数**：
  | 参数 | 类型 | 默认值 | 可选值 | 说明 |
  | :--- | :--- | :--- | :--- | :--- |
  | `mode` | 字符串 | 空（跟随面板设置） | `multicast`, `unicast`, `http` | 强制指定播放流模式 |
  | `fmt` | 字符串 | `m3u` | `m3u`, `txt`, `json` | 导出格式，默认输出标准 M3U |
  | `udpxy` | 字符串 | 空 | 如 `192.168.1.1:4022` | 临时覆盖组播转发器 host:port |
  | `scheme` | 字符串 | 空 | `rtp`, `udp`, `igmp` | 强制改写组播协议头 |
  | `xteve` | 字符串 | 空 | `true` | 输出兼容 xTeVe / Plex 的 `udp://@...` 地址 |
  | `all` | 字符串 | 空 | `true` | 是否包含购物频道等过滤内容 |

#### 响应示例 (M3U 格式)

```m3u
#EXTM3U x-tvg-url="http://192.168.1.100:8888/api/epg" catchup="default"
#EXTINF:-1 tvg-id="1" tvg-name="CCTV-1" tvg-logo="http://192.168.1.100:8888/logos/cctv1.png" tvg-chno="1" group-title="央视频道" catchup="default" catchup-days="7" catchup-source="http://192.168.1.100:8888/api/play?id=1&mode=unicast&utc=${start}&lutc=${end}" timeshift="7",CCTV-1 综合
http://192.168.1.100:8888/api/play?id=1&mode=unicast
```

#### M3U 头部与标签规范：
- `#EXTM3U x-tvg-url="..." catchup="default"`：声明全局 EPG 地址与回看能力，绝大多数现代播放器（如 TiviMate / Televizo / APTV）可全自动配置 EPG 与时移。
- `tvg-id` / `tvg-chno`：频道数字唯一标识与台号。
- `catchup="default"`：指定回看实现类型。
- `catchup-days="7"` / `timeshift="7"`：声明支持 7 天时移回看。
- `catchup-source`：回看动态模板地址，播放器在请求回看时会将 `${start}` 与 `${end}` 替换为具体的时间戳。

---

### 2.2 统一流媒体与回看重定向 (`/api/play`)

万能播放与回看转发中心。根据请求参数计算时间区间、请求运营商鉴权或构建 RTSP CST 播放参数，最后返回 `302 Found` 临时重定向至真实的播放地址。

- **请求方法**：`GET`, `HEAD`（`HEAD` 请求仅返回 HTTP 响应头，适用于播放器连通性探测）
- **请求 URL**：`http://<server-ip>:8888/api/play`
- **查询参数**：

  **频道标识参数 (至少提供一个)**：
  - `id`: 频道数字 ID（推荐，如 `id=1`）
  - `channel`: 频道名称（如 `channel=东方卫视`，支持自动别名识别）
  - `channel_id`: 别名，同 `id`

  **播放模式参数**：
  - `mode`: `unicast`（单播/RTSP 回看）或 `http`（运营商 HTTP 动态流），缺省时自动智能判断。

  **回看时间参数 (兼容所有主流播放器格式)**：
  - 开始时间（依次匹配）：`start`, `utc`, `begin`, `time`, `timestamp`
  - 结束时间（依次匹配）：`end`, `lutc`, `utcend`, `lutcend`, `stop`
  - 节目时长（自动累加计算）：`duration`, `dur`（秒数，若缺少结束时间，则 `end = start + duration`）
  - 范围参数：`playseek=YYYYMMDDHHMMSS-YYYYMMDDHHMMSS` 或 `playseek=1726915200-1726918800`

#### 支持的时间格式说明：
| 格式示例 | 长度/类型 | 适用场景 |
| :--- | :--- | :--- |
| `1726915200` | 10位纯数字 | Unix 秒级时间戳（TiviMate `{utc}`、Televizo `${start}`） |
| `1726915200000` | 13位纯数字 | Unix 毫秒级时间戳（部分安卓或 Web 客户端） |
| `20260921120000` | 14位纯数字 | 中国标准时间 CST (UTC+8) 紧凑格式 `YYYYMMDDHHMMSS` |
| `202609211200` | 12位纯数字 | 紧凑时间格式 `YYYYMMDDHHMM` |
| `2026-09-21T12:00:00+08:00` | ISO 字符串 | 标准 RFC 3339 / ISO 8601 日期时间 |

#### 直播时移（重新从头播放进行中的节目）：
当用户选择当前正在播出的节目“从头播放”时，开始时间为过去时间，而结束时间通常落在未来。服务自动将结束时间限制在当前系统时间（+5分钟冗余），确保 RTSP 服务端与 HTTP 门户正常响应，不抛出越界错误。

#### 响应状态码：
- `302 Found`: 重定向成功，`Location` 头为真实音视频流地址。
- `400 Bad Request`: 参数不合法，如回看时间超出频道设定的最大回看天数、时间格式无效或结束时间早于开始时间。
- `404 Not Found`: 频道不存在或未启用。
- `502 Bad Gateway`: 无法从运营商门户获取播放流。
- `503 Service Unavailable`: 未配置或未启用 IPTV 账号认证。

---

### 2.3 XMLTV 电子节目单 (`/api/epg`)

输出符合标准 XMLTV 协议的中文电视节目预告与回顾数据。

- **请求方法**：`GET`, `HEAD`
- **请求 URL**：`http://<server-ip>:8888/api/epg`
- **查询参数**：
  - `daysAgo`: 整数，过滤的历史节目天数（例如 `daysAgo=7` 表示包含过去 7 天内至今的节目）
- **返回内容类型**：`application/xml; charset=utf-8`
- **特性**：
  - 自动频道去重（CCTV-1、CCTV-1 高清等共享同一节目数据源）
  - 标准时区偏移声明（`+0800`）
  - 完整支持节目详情/简介（`<desc lang="zh">...</desc>`），当源端提供描述或剧情概要时自动注入，兼容 TiviMate、Televizo、APTV 等播放器节目简介展示
  - 内存高速缓存与定时更新机制

---

### 2.4 内嵌专用网页播放器 (`/player`)

系统内置专为 IPTV 直播打造的现代化 Web 网页播放器，100% 静态资产内嵌编译，在隔离的 IPTV 专网/内网环境下无需任何外网 CDN 即可完整流畅运行。

- **访问地址**：`http://<server-ip>:8888/player` 或 `http://<server-ip>:8888/player.html`
- **核心特性**：
  - **双引擎无缝播放**：内置 MSE MPEG-TS 解复用器（`mpegts.js`）与 HLS 流引擎（`hls.js`），原生支持 HTTP 连续 MPEG-TS 组播流（如 `http://<host>/rtp/233.18.204.215:5140?fcc=...`）、HLS 直播流与 HTML5 原生播放。
  - **自定义流地址播放**：支持通过弹窗或 URL 参数直接播放任意流媒体地址（如 `/player?url=http%3A%2F%2Ftvpanel.netioe.com%2Frtp%2F233.18.204.215%3A5140%3Ffcc%3D124.75.26.151%253A15970`）。
  - **内置流媒体中继代理 (`/api/stream/proxy`)**：针对上游组播网关或 IPTV 代理未配置 CORS 跨域响应头的情况，提供高性能服务端中继代理，解决浏览器跨域拦截与专网隔离问题。
  - **EPG 时间线与时移回看**：自动从 `/api/epg/programmes` 加载节目单并展示当前播出状态，历史节目支持一键回看（Catchup），正在直播节目支持进度显示。
  - **遥控器与键盘友好**：支持方向键换台（`↑`/`↓`）、音量调节（`←`/`→`）、数字键选台、全屏切换（`F`）、静音（`M`）、频道抽屉（`C`）、节目单抽屉（`E`）、自定义流地址（`U`）。
  - **多模式即时切换**：支持在播放器内一键切换组播（Multicast）、运营商官方 HTTP 直播及 RTSP 单播。

---

### 2.5 兼容性与根路径接口

- `GET /playlist.m3u`: 根路径标准 M3U 播放列表（自动对接网页播放器与第三方播放器）。
- `GET /epg.xml`, `GET /epg.xml.gz`: 根路径 XMLTV 电子节目单（标准及压缩格式）。
- `GET /api/tsM3u8`: 兼容旧版项目单播 M3U 导出。
- `GET /api/schedule`: 触发后台 EPG 增量同步调度。

---

## 3. 管理面板 API (`/api/panel/...`)

管理面板 API 用于 Web UI 前端交互、自动化运维与脚本控制。

### 3.1 鉴权机制

- **本地回环白名单**：来自 `127.0.0.1` / `::1` 的连接默认放行（便于本地调试）。
- **远程鉴权**：非本地连接需在请求头携带管理员令牌：
  ```http
  Authorization: Bearer <IPTV_PANEL_TOKEN>
  ```
  管理员令牌通过启动环境变量 `export IPTV_PANEL_TOKEN="你的安全密码"` 配置。

---

### 3.2 系统状态与设置

#### 1. 查询系统状态
- **URL**：`GET /api/panel/status`
- **返回**：频道总数、已启用数、扫描任务状态、EPG 状态、IPTV 认证有效性等。

#### 2. 获取配置信息
- **URL**：`GET /api/panel/settings`
- **返回**：包含 `forward`、`iptv`、`logos`、`scan`、`ai`、`group_order`、`group_channel_order` 等配置。AI API Key 在返回时会自动脱敏。

#### 3. 保存配置信息
- **URL**：`PUT /api/panel/settings`
- **请求体**：
  ```json
  {
    "settings": {
      "forward": {
        "address": "192.168.190.1:4022",
        "protocol": "rtp",
        "play_mode": "unicast",
        "fcc": "124.75.26.151:15970",
        "catchup_template": "&utc=${start}&lutc=${end}"
      },
      "iptv": {
        "enabled": true,
        "uid": "153xxxxxxxx",
        "sn": "...",
        "mac": "...",
        "ip": "192.168.1.10",
        "type": "B860A",
        "auth_host": "222.68.208.73:7001"
      },
      "group_order": ["央视频道", "卫视频道", "上海频道", "数字频道", "其它", "待识别"]
    },
    "clear_api_key": false
  }
  ```

---

### 3.3 频道管理与探测

#### 1. 获取频道列表
- **URL**：`GET /api/panel/channels`
- **返回**：完整频道列表数组。

#### 2. 更新频道属性
- **URL**：`PUT /api/panel/channels/{id}`
- **请求体**：可更新 `name`, `group`, `logo`, `unicast_url`, `operator_id`, `catchup_days`, `enabled` 等。

#### 3. 实时画面截取
- **URL**：`GET /api/panel/channels/{id}/snapshot`
- **返回**：`image/jpeg` 格式的实时电视画面截图。

#### 4. AI 视觉台标识别
- **URL**：`POST /api/panel/channels/{id}/identify`
- **功能**：调用配置的大模型视觉接口识别截图中的电视台名称与台标。

#### 5. 单频道分辨率探测
- **URL**：`POST /api/panel/channels/{id}/probe`
- **功能**：通过 FFmpeg 探测视频流的真实分辨率（如 `1080p`, `4K`, `720p`）并持久化保存。

---

### 3.4 频道模糊匹配与关键词映射

#### 1. 频道名模糊匹配搜索
- **URL**：`GET /api/panel/channels/match?name={name}`
- **示例**：`GET /api/panel/channels/match?name=cctv16`
- **返回**：匹配得分最高的目标（如 `CCTV-16HD`）及候选列表。

#### 2. 获取与保存关键词映射
- `GET /api/panel/mappings`: 获取自定义的频道名称映射表。
- `POST /api/panel/mappings`: 增加或更新映射（如 `{"keyword":"cctv16","target_id":"51","target_name":"CCTV-16HD"}`）。
- `DELETE /api/panel/mappings/{keyword}`: 删除映射。

---

### 3.5 组播扫描与批量探测

- `POST /api/panel/scan/start`: 启动网段组播探测任务。
- `POST /api/panel/scan/stop`: 停止当前扫描。
- `POST /api/panel/probe/start?only_missing=true|false`: 启动全量或仅针对无分辨率频道的批量分辨率后台探测。
- `POST /api/panel/probe/stop`: 终止探测任务。
- `GET /api/panel/probe/status`: 查询当前批量探测的进度指标（已探测、总数、运行状态）。

---

### 3.6 台标管理与本地缓存

- `GET /api/panel/logos/status`: 查询配置的所有台标源同步状态与台标总数。
- `POST /api/panel/logos/refresh`: 从 GitHub / HTTP 源拉取最新台标索引。
- `POST /api/panel/logos/match`: 匹配指定频道的台标候选（优先亮色、特定地域）。
- `POST /api/panel/logos/download`: 将选定的台标图片下载并固化到本地服务。
- `GET /logos/{file...}`: 读取本地缓存的台标图片。

---

### 3.7 EPG 节目管理

- `GET /api/panel/epg/status`: EPG 缓存状态、收录频道与节目条数。
- `POST /api/panel/epg/refresh`: 手动触发一次 EPG 节目单同步。
- `GET /api/panel/epg/programmes?id={id}&date={YYYY-MM-DD}`: 获取特定频道特定日期的节目单详情（包含节目标题、起止时间、详情简介 `desc` 等）。

---

## 4. 主流播放器实战配置指南

### 4.1 TiviMate 配置 (推荐)

TiviMate 是 Android TV / 电视盒子上体验最好的播放器之一，对本服务的组播、单播与时移回看提供原生完美支持。

1. **添加播放列表**：
   - 打开 TiviMate -> 设置 -> 播放列表 -> 添加播放列表 -> 选择 **M3U 播放列表**。
   - 输入播放列表 URL：
     ```text
     http://<你的服务器IP>:8888/api/playlist?mode=unicast
     ```
     *(若局域网内所有设备均支持组播且路由配置了 udpxy，也可使用 `mode=multicast`)*。
2. **配置 EPG**：
   - 因为 M3U 文件头已包含 `x-tvg-url="http://<服务器IP>:8888/api/epg"`，TiviMate 通常会**自动绑定 EPG**。
   - 若未自动绑定，在“EPG 来源”中手动添加：
     ```text
     http://<你的服务器IP>:8888/api/epg
     ```
3. **启用回看 (Catchup)**：
   - 播放列表选项中，找到 **回看 (Catchup)** 设置项。
   - **回看类型**：选择 **默认 (Default)**。
   - **回看天数**：设置为 `7` 天（或自动跟随频道元数据）。
   - **效果**：在节目单界面，过去 7 天内的节目均会带有圆形回看图标，点击即可直接无缝回放并支持拖动进度条。

---

### 4.2 Televizo 配置 (推荐)

Televizo 是手机、平板与电视端兼顾的专业 IPTV 播放器。

1. **添加播放列表**：
   - 打开 Televizo -> 设置 -> 播放列表 -> 点击右上角 `+` -> 选择 **新建 M3U 播放列表**。
   - 名称：随意填写（如“电信IPTV”）。
   - 播放列表链接：
     ```text
     http://<你的服务器IP>:8888/api/playlist?mode=unicast
     ```
2. **添加 EPG 电子节目指南**：
   - 在播放列表设置下方，找到 **EPG 代码 (EPG 链接)**。
   - 填写：
     ```text
     http://<你的服务器IP>:8888/api/epg
     ```
3. **回看设置 (Archive)**：
   - **归档类型 (Archive Type)**：选择 **默认 (Default)**。
   - **回看模板**：本服务已在 M3U 中预置 `&utc=${start}&lutc=${end}`，Televizo 默认直接识别，无需任何额外配置即可一键回看。

---

### 4.3 APTV (Apple TV / iOS / macOS)

1. 打开 APTV，点击右上角添加配置，选择 **添加订阅链接**。
2. 订阅 URL 填写：
   ```text
   http://<你的服务器IP>:8888/api/playlist?mode=unicast
   ```
3. APTV 会自动解析 M3U 中的 `x-tvg-url` 获取 EPG，并根据 `catchup` 标签激活时移回看。

---

### 4.4 DIYP / 影音壳子 / 影视仓

适用于国内各类通用电视盒子应用。

1. 在应用设置的“接口地址 / 自定义源”中填入：
   ```text
   http://<你的服务器IP>:8888/api/playlist?fmt=txt&mode=unicast
   ```
2. 在“EPG 接口”填入：
   ```text
   http://<你的服务器IP>:8888/api/epg
   ```

---

### 4.5 PotPlayer / VLC (PC 电脑端)

1. 打开 PotPlayer / VLC，按快捷键 `Ctrl + U`。
2. 输入播放列表地址：
   ```text
   http://<你的服务器IP>:8888/api/playlist?mode=unicast
   ```
3. 播放器将载入完整频道列表，点击任意频道即可流畅播放。

---

## 5. 回看与时移技术细节规范

### 5.1 模板参数体系

在面板“IPTV配置” -> “播放与转发”卡片中，提供了可视化的“回看模板参数”配置：

| 模板预设 | 适用播放器 | 变量替换规则 |
| :--- | :--- | :--- |
| `&utc=${start}&lutc=${end}` (默认推荐) | Televizo / DIYP / 国内多数应用 | `${start}` = 起始秒级时间戳，`${end}` = 结束秒级时间戳 |
| `&start={utc}&end={utcend}` | TiviMate 标准 | `{utc}` = 起始秒级时间戳，`{utcend}` = 结束秒级时间戳 |
| `&utc={utc}&lutc={utcend}` | 跨客户端通用兼容 | 兼容各种对参数名有偏好的播放器 |
| `&start=${timestamp}&duration=${duration}` | 时长格式播放器 | `${timestamp}` = 开始时间，`${duration}` = 持续总秒数 |
| 自定义输入… | 特殊播放器或反代网关 | 允许自由指定任意 query 参数，后台自动补齐与编码 |

### 5.2 RTSP 上海电信 CST 时间转换

上海电信 RTSP 单播服务要求回看时间参数使用 `playseek=YYYYMMDDHHMMSS-YYYYMMDDHHMMSS` 格式，且**严格基于北京时间 (China Standard Time, UTC+8)**。

当 `/api/play` 接收到各类时间参数时：
1. 无论请求参数是 Unix 秒级时间戳、毫秒戳还是 ISO 8601 字符串，服务均首先将其转换为 Go 标准 `time.Time`。
2. 随后将其强制转换至 `time.FixedZone("CST", 8*3600)`。
3. 格式化为电信机顶盒所要求的 `20060102150405` 形式，拼装为目标 RTSP 的 URL 参数，并执行 302 重定向。
4. 客户端无需关心时区与格式转换，实现跨平台无缝兼容。

---

## 6. 常见问题排查 (FAQ)

#### Q1: 为什么在 TiviMate / Televizo 中看不到回看节目？
- **原因 1**：所选频道未设置单播地址或未开启回看天数。请在 Web 管理面板编辑对应频道，确保“回看天数”大于 0（通常为 7 天），且拥有单播地址或运营商 ID。
- **原因 2**：EPG 尚未同步完成。请在面板中检查“EPG 管理”是否已有节目单，若为空可点击“同步 EPG 节目”。

#### Q2: 点击正在播出的节目“从头播放”提示 400 错误？
- 本服务已内置**直播时移保护**：当检测到节目的结束时间在当前系统时间之后时，会自动将结束时间修正为当前时间，避免 RTSP 服务端因请求“未来时间”而报错。

#### Q3: 为什么局域网内无法播放组播流（`mode=multicast`）？
- 组播流依赖路由器开启 **IGMP Snooping** 以及配置了 **udpxy** 服务。若路由器未部署相关服务，请直接在播放器中使用单播模式：`http://<IP>:8888/api/playlist?mode=unicast`。

#### Q4: 如何在反向代理（如 Nginx / Caddy）后使用本服务？
- 建议配置 `X-Forwarded-Host` 与 `X-Forwarded-Proto` 头。
- 对于管理面板，务必设置 `export IPTV_PANEL_TOKEN="强密码"`，保护管理 API 安全。
