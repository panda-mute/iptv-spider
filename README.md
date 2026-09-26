# 上海电信 IPTV 控制台

Go 编写的上海电信 IPTV 频道、EPG 与组播管理服务，内置中文 Web 面板，无需 Node.js 或外部前端资源。支持 `igmp://` 组播频道、RTSP 单播包装、运营商 HTTP 直播/回看以及 `funcportalAuth` 门户认证。

---

## 快速开始

### 方式一：Docker 部署（推荐）

预构建多架构镜像（`linux/amd64` 与 `linux/arm64`）支持在轻量级 NAS、软路由或 VPS 上开箱即用。

#### 1. 使用 Docker Compose

创建 `compose.yaml` 文件：

```yaml
services:
  iptv:
    image: ghcr.io/panda-mute/iptv-spider:latest # 或使用 build: . 本地构建
    container_name: iptv-spider
    restart: unless-stopped
    ports:
      - "8888:8888"
    # 如果宿主机需要直接绑定网卡接收组播或与内网 IPTV 转发器通信，可选用 host 网络：
    # network_mode: host
    environment:
      - IPTV_PANEL_TOKEN=请替换为强随机访问令牌 # 远程访问面板时必须设置
      - TZ=Asia/Shanghai # 时区（保证 EPG 与回看时间对齐北京时间 CST）
      - IPTV_DATA_DIR=/app/data # 数据持久化目录（默认 /app/data）
    volumes:
      - ./data:/app/data # 核心数据目录（存储 panel.json、epg.json 及自动释放的本地台标）
      - ./config.yaml:/app/config.yaml:ro # 基础设施配置文件（只读挂载）
```

启动容器：

```bash
docker compose up -d
```

#### 2. 使用 Docker CLI

```bash
docker run -d \
  --name iptv-spider \
  --restart unless-stopped \
  -p 8888:8888 \
  -e IPTV_PANEL_TOKEN="请替换为强随机访问令牌" \
  -e TZ="Asia/Shanghai" \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/panda-mute/iptv-spider:latest
```

#### 3. Docker 部署参数说明

- **网络模式 (Bridge 模式 vs Host 模式)**：
  - **端口映射 (Bridge - 默认推荐)**：使用 `-p 8888:8888`，适合由软路由或专用设备运行 `udpxy` / `msr` 组播转发器（如 `192.168.x.x:4022`）的典型组网环境。
  - **Host 模式 (`network_mode: host`)**：当宿主机本身直接接入 IPTV 专网 VLAN、需要直连光猫或接收特定网卡组播数据包时，推荐启用 host 网络模式。
- **存储卷与数据持久化**：
  - `./data:/app/data`：保存配置 `panel.json`（频道列表、扫描结果、自定义关键词映射）、`epg.json`（节目单缓存）及 `logos/` 目录。
  - **本地台标自动释放**：服务首次启动时，会自动将内置的精简高清/4K/央视/卫视/本地台标解压到 `/app/data/logos` 目录中且仅执行一次。频道默认优先匹配本地台标，零网络依赖且加载速度极快。
- **环境变量一览**：
  | 环境变量 | 默认值 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | `IPTV_PANEL_TOKEN` | 空 | 推荐 | 面板访问令牌。非本机访问必须配置；留空时管理 API 仅允许回环访问 (`localhost` / `127.0.0.1`) |
  | `TZ` | `Asia/Shanghai` | 是 | 容器时区，确保 EPG 节目单与 RTSP CST 时间戳解析正确 |
  | `IPTV_DATA_DIR` | `/app/data` | 否 | 数据存储路径 |

---

### 方式二：二进制直接运行

需要 Go 1.24+ 运行环境；截图与视频流探测功能依赖系统已安装 `ffmpeg`。

```bash
# 编译二进制文件
go build -trimpath -o bin/iptv-spider .

# 设置安全访问令牌并启动
export IPTV_PANEL_TOKEN='请替换为强随机访问令牌'
./bin/iptv-spider -c config.yaml
```

启动后访问 `http://<服务器IP>:8888/`，输入配置的令牌即可进入管理控制台。

---

## 核心功能与使用指南

### 1. IPTV 账号与运营商同步

1. 进入“账号与转发”，填写 IPTV 业务账号、SN、MAC、机顶盒认证 IP、型号及认证服务器地址，保存并启用认证。
2. 机顶盒认证 IP 填写点分 IPv4（如 `10.x.x.x`），服务自动转换规范化运营商格式（如 `010,xxx,xxx,xxx`），作为运营商门户认证传参。
3. 在具备 IPTV 专网网络连通性的环境下，点击“同步运营商频道”，服务将自动完成认证并拉取官方分类频道与回看配置。
4. 频道采用独立覆盖层存储，后续同步或重复扫描不会覆盖用户自定义的频道名称、数字台号、分组和台标。

### 2. 组播转发与扫描

- **转发器对接**：本项目对接局域网内已有的组播转发服务（如 udpxy / msr），不内置 RTP/FCC 引擎。可在面板中配置组播转发器地址（如 `192.168.1.1:4022`）及 FCC 服务器。
- **组播地址自动适配**：默认组播扫描范围为 `233.18.204.1` - `233.18.204.254`（端口 `5140`）。
- **真实 MPEG-TS 探测**：探测过程中通过转发器拉取流数据，仅当连续捕获到合法的 MPEG-TS 同步字节（`0x47`）时才确认为有效电视频道，避免将 HTTP 200 错误页面误判为视频流。

### 3. 频道管理与关键词映射

- **默认预设关键词映射**：
  - 针对运营商命名与日常标准习惯的差异，系统内置了默认映射规则（如 `体育频道` -> `五星体育`、`卡酷卡通` -> `卡酷少儿` 等）。
  - 所有映射均作为用户数据保存在 `panel.json` 中，**非代码硬编码**。用户可在控制台自由修改、新增或删除关键词映射，并支持一键“恢复默认预设”。
  - 导入频道、扫描发现及加载列表时，会自动应用映射规则，统一修正频道名称、台标与分组。
- **分组预分类**：内置标准分组体系（4K、央视、卫视、高清、本地、少儿、标清等），根据频道特性与组播网段自动归类。

### 4. 本地预设台标与多台标源

- **内置本地预设台标**：精选并内置常用央视、卫视、本地及 4K/高清频道高质量台标（无外网请求、无额外哈希、开箱即用）。
- **本地优先匹配**：所有频道在解析台标时优先匹配本地 `logos/` 目录；用户亦可随时在面板中上传自定义台标图片。
- **多台标源热备**：在“智能识别”中支持配置多个外部 GitHub/API 台标仓库作为后备源，系统自动按顺序优先匹配常规亮色台标，深色台标自动作为后备。

### 5. 单播与回看（主流播放器原生兼容）

系统深度优化并原生兼容 **TiviMate**、**Televizo**、**APTV** 等主流播放器的回看（Catchup / Archive / Timeshift）功能：

- **M3U 规范标签**：
  - `#EXTM3U` 头部自动输出 `x-tvg-url` 与 `catchup="default"`。
  - `#EXTINF` 标签输出标准 `catchup="default"`、`catchup-days="7"`、`timeshift="7"` 以及数字台号 `tvg-chno="1"`。
- **回看动态模板**：Web 控制台支持可视化切换与配置回看模板（如 `&utc=${start}&lutc=${end}`、`&start={utc}&end={utcend}` 等）。
- **全方位变量解析**：完美支持 Unix 秒级时间戳（`{utc}`）、毫秒戳、Televizo 变量（`${start}` / `${end}` / `${timestamp}` / `${duration}`）。
- **北京时间 CST 严格对齐**：针对上海电信 RTSP 回看要求的 `playseek=YYYYMMDDHHMMSS-YYYYMMDDHHMMSS` 格式，服务端全自动完成时区与紧凑时间格式转换。
- **直播时移保护**：对正在直播节目的“从头播放”请求，自动识别并容错未来结束时间，平滑时移。
- **支持 HEAD 请求**：全面兼容 ExoPlayer 与 VLC 内核在流预探测时的探测请求。

### 6. EPG 电子节目单管理

- **轻量原子存储**：默认无需配置 MySQL，以原子写入方式保存于 `data/epg.json`。同时也支持通过 `config.yaml` 开启 MySQL 持久化。
- **自动定时同步**：支持自定义定时同步间隔（默认每 6 小时）、历史天数与未来天数。
- **标准 XMLTV 导出**：提供标准 `/api/epg` 接口，支持按频道、按日期检索节目。

### 7. 智能识别与一键探测

- **画面识别**：支持配置 OpenAI 兼容的视觉模型 API（如 GPT-4o），后台通过 `ffmpeg` 截取未知频道画面并调用视觉模型提供频道与分组识别建议。
- **模型台标匹配**：纯文本向模型发送频道信息与台标目录，智能匹配最相近的标准台标名称。
- **一键探测视频规格**：支持在后台批量探测频道的视频分辨率（4K / 1080P / 720P），实时显示进度并打上分辨率标识徽章。
- **组内频道排序**：支持在 Web 界面对任意分组内的频道进行绝对次序调整、拖拽重排或一键排序。

---

## 常用接口速查

> 📖 **完整 API 详细参考与各播放器实战配置指南，请参阅专用文档：[docs/api.md](docs/api.md)**

| 接口 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/` | `GET` | 中文 Web 管理控制台 |
| `/player` | `GET` | 内嵌专用 Web 网页播放器（全离线 MSE/HLS 播放、MPEG-TS 直播、时移与遥控交互） |
| `/playlist.m3u` | `GET`, `HEAD` | 根路径标准 M3U 播放列表 |
| `/epg.xml` | `GET`, `HEAD` | 根路径 XMLTV 电子节目单 |
| `/api/playlist?mode=multicast` | `GET`, `HEAD` | 组播 M3U 播放列表（默认） |
| `/api/playlist?mode=unicast` | `GET`, `HEAD` | RTSP 包装单播与 7 天时移回看 M3U |
| `/api/playlist?mode=http` | `GET`, `HEAD` | 运营商官方 HTTP/HLS 直播与回看 M3U |
| `/api/m3u8` | `GET`, `HEAD` | 兼容旧版 M3U 订阅地址 |
| `/api/epg` | `GET`, `HEAD` | 标准 XMLTV 电子节目单 |
| `/api/play` | `GET`, `HEAD` | 统一流媒体路由与回看重定向 |
| `/api/panel/channels` | `GET` | 频道完整管理列表（需 Token） |
| `/api/panel/mappings` | `GET`, `POST` | 频道关键词映射配置（需 Token） |
| `/api/panel/mappings/reset` | `POST` | 一键恢复默认预设关键词映射（需 Token） |

---

## 目录结构说明

- `modules/panel/`：Web 控制台、数据存储、频道映射、扫描与台标服务。
- `modules/spider/`：运营商 IPTV 协议客户端与认证解析引擎。
- `modules/m3u/`：M3U / TXT / JSON 播放列表生成与规范化。
- `router/`：Web 路由与公共/管理 API 接口定义。
- `data/`：核心持久化数据目录（建议挂载）：
  - `panel.json`：系统设置、频道覆盖层及自定义关键词映射。
  - `epg.json`：电子节目单数据。
  - `logos/`：本地频道台标文件库。

---

## 自动化测试与构建

```bash
# 运行单元测试
make test

# 编译二进制程序
make build
```

---

## 参考与致谢

本项目在架构设计与功能实现过程中，参考了以下优秀的开源项目，在此表示诚挚的感谢：

- [denymz/sh-tel-iptv-spider](https://github.com/denymz/sh-tel-iptv-spider)：提供了上海电信 IPTV 频道抓取、EPG 生成与组播扫描的重要基础实现与思路。
- [stackia/rtp2httpd](https://github.com/stackia/rtp2httpd)：提供了轻量高效的 RTP 组播转 HTTP、FCC 快速切台以及 Web 播放器设计参考。
