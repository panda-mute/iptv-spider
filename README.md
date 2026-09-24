# 上海电信 IPTV 控制台

Go 编写的上海电信 IPTV 频道、EPG 与组播管理服务，内置中文 Web 面板，无需 Node.js 或外部前端资源。认证流程参考原项目与 `shdx.php`，支持 `igmp://` 频道和新版 `funcportalAuth` 门户。

## 启动

需要 Go 1.24+；截图与识别需要 `ffmpeg`。运营商同步通过系统默认网络访问认证服务器，使用有效的机顶盒身份信息。

```bash
go build -o bin/iptv-spider .
# 远程访问面板时必须设置令牌；请自行生成足够长的随机值。
export IPTV_PANEL_TOKEN='替换为随机访问令牌'
./bin/iptv-spider -c config.yaml
```

访问 `http://服务器IP:8888/`，输入上述令牌。未设置令牌时，管理 API 仅接受本机回环连接。反向代理部署时，也应设置令牌，并使用 HTTPS。M3U/TXT/JSON 播放列表是供播放器使用的公开只读接口，请按需要限制服务的网络访问范围。

默认不连接 MySQL，不启用对象存储，首次启动不会使用示例账号发起认证。账号、转发、扫描和 AI 配置在面板保存后生效；`config.yaml` 中的基础设施配置需要重启。

也可以在 Linux 使用 Docker：

```bash
export IPTV_PANEL_TOKEN='替换为随机访问令牌'
docker compose up -d --build
```

Compose 使用普通容器网络，将面板端口 `8888` 映射到宿主机，无需网络管理权限。配置和频道保存在挂载的 `./data` 中。

## IPTV 账号

1. 在“账号与转发”填写 IPTV 账号、SN、MAC、机顶盒认证 IP、型号及认证服务器，然后启用认证。
2. 机顶盒认证 IP 使用原身份配置中的点分 IPv4，客户端自动转为 `030,001,002,003` 一类的运营商格式。此字段是认证参数，不是本机出口 IP。
3. 保存后回到频道管理，点击“同步运营商频道”。状态区显示进度或错误。

认证使用系统默认路由，不绑定网卡或源 IP，不继承 HTTP 代理。本机已通过真实认证测试并获取 168 个频道。面板不再提供 VLAN、DHCP 或路由管理；旧版保存的专网配置会在加载时自动移除，账号、频道和 API key 保留。旧版未填写转发地址时，自动补为 `192.168.190.1:4022`；已有自定义转发地址保留。

## 转发与 FCC

本项目对接已有组播转发服务，不内置 RTP/FCC 转发引擎。默认转发服务为 `192.168.190.1:4022`，可在面板修改。默认路径为：

```text
http://192.168.190.1:4022/rtp/239.45.0.1:5140?fcc=124.75.26.151:15970
```

默认 FCC 取自 PHP 参考，可修改或清空；URL 中的冒号可能按标准查询参数编码为 `%3A`。已带有 FCC 的频道保留其值。导出、扫描、截图与识别共用同一套地址生成逻辑，支持选择 `/udp/` 以兼容其他转发服务。转发器负责接收运营商组播流，本服务通过 HTTP 调用它。

## 频道管理与扫描

- 同步按运营商分类获取频道；支持编辑频道 ID、名称、`group`、`logo`、源 URL 和启用状态。修改 ID 后保留与原运营商频道的关联，再次同步不会生成重复频道；重复 ID 会提示冲突。分组可下拉选择已有分组或输入自定义名称，填写台标 URL 后可预览图片。
- 手动配置作为独立覆盖层保存，不会被后续运营商刷新或重复扫描覆盖。停用的频道不再导出。
- 扫描支持起止组播 IPv4、起止端口、并发数和超时；每次最多 65,536 个组合，并发 1–32，超时 1–30 秒。只扫描自己使用的 IPTV 网段。
- 探测通过 HTTP 转发器进行，只有收到重复同步字节的 MPEG-TS 流才计为发现，HTTP 200 或 HTML 错误页面不算有效频道。
- 扫描任务可停止，状态会显示已探测、发现数量和最近一次失败信息。扫描结果立即持久化；重启保留频道，运行中的扫描任务不自动恢复。

## 未知频道识别

在“智能识别”配置 OpenAI 兼容 API Base URL（如 `https://api.example.com/v1`）、API key 和支持图片输入的模型 ID。也支持填写完整 `/chat/completions` 地址。

在频道列表点击“截图”检查画面，点击“识别”会用 `ffmpeg` 截取 JPEG 并通过 Chat Completions 的 `image_url` Base64 数据发送给配置的 API。需要服务商支持该图片输入形式；纯文本模型无法识别。请求会产生相应服务商费用。

识别返回频道名、分组、置信度和依据，仅保存为建议。点击“填入识别建议”，检查后“保存频道”才会应用；不确定时保留未知频道。台标默认主源为 https://github.com/sggc/SDU-IPTV-PRO/tree/main/logo ，支持配置多个台标源按优先级自动匹配（深色台标自动作为后备）；手动填写的 URL 优先，清空后恢复自动匹配。服务内置主源目录快照，启动时刷新目录，源不可用时仍可使用快照或上次保存的目录；图片加载仍需要访问台标源。台标地址不由模型猜测。

API key 不会从管理 API 回显，留空保留原值，勾选清除可删除。key 以明文保存在权限为 `0600` 的本地数据文件中，应妥善保管备份。图片仅用于本次请求，不写入频道存储。

接口格式参考：[OpenAI 图片与视觉文档](https://developers.openai.com/api/docs/guides/images-vision)。

## 接口与兼容性

> 📖 **完整 API 详细参考与各播放器实战配置指南，请参阅专用文档：[docs/api.md](docs/api.md)**

| 接口 | 说明 |
| --- | --- |
| `GET /` | 管理面板 |
| `GET /api/m3u8` | M3U，保留 `udpxy`、`scheme`、`xteve=true`、`all=true` 参数 |
| `GET /api/playlist?fmt=m3u` | `fmt` 可选 `m3u`、`txt`、`json`，配置立即反映 |
| `GET /api/epg?daysAgo=1` | XMLTV 节目单，本地文件/MySQL 两种模式均支持 |
| `GET /api/tsM3u8` | 单播 M3U（兼容旧路径），无需 MySQL |
| `GET /api/playlist?mode=http` | 运营商 HTTP/HLS 直播及 HTTP 回看 M3U |
| `GET /api/playlist?mode=unicast` | RTSP 经转发器输出 HTTP 的单播/回看 M3U |
| `GET /api/play?id=51&mode=http` | 实时认证取得 HTTP 直播地址，302 跳转 |
| `GET /api/play?id=51&mode=http&start=…&end=…` | HTTP 回看；开始/结束时间为 Unix 秒 |
| `GET/PUT /api/panel/settings` | 设置；PUT 为 `{"settings":{...},"clear_api_key":false}` |
| `GET /api/panel/channels` | 合并运营商数据和手动覆盖后的频道 |
| `PUT /api/panel/channels/{id}` | 完整保存频道编辑 |
| `POST /api/panel/refresh` | 异步同步频道 |
| `POST /api/panel/epg/refresh` | 异步同步 EPG，两种存储模式一致 |
| `GET /api/panel/epg/status` | EPG 进度、更新时间、节目数量和存储模式 |
| `GET /api/panel/epg/programmes?id=51&date=2026-09-20` | 指定频道和日期的节目单 |
| `POST /api/panel/scan/start`、`POST /api/panel/scan/stop` | 扫描控制 |
| `GET /api/panel/status` | 扫描和同步状态 |
| `GET /api/panel/channels/{id}/snapshot` | 截图 JPEG |
| `POST /api/panel/channels/{id}/identify` | 识别并保存建议 |

管理接口使用 `Authorization: Bearer <IPTV_PANEL_TOKEN>`，请求体为 JSON。原有无鉴权、用 GET 触发写操作的 `/api/run` 返回 410，改用面板 POST 接口。频道 ID 作为路径参数时需要 URL 编码。

默认 `system.db-type: none`：节目单存入 `data/epg.json`，无需数据库。改为 `system.db-type: mysql` 并填写 `mysql.path`、`db-name`、`username`、`password` 等连接配置后重启：新 EPG 存入 `panel_epg_snapshots` 表，同一 `/api/epg` 和管理接口继续使用。MySQL 连接失败会明确停止启动，避免悄悄退回文件模式。两模式不会自动互迁已有 EPG，切换后点击同步即可重新生成；面板设置和频道覆盖仍使用 `data/panel.json`。原 MySQL 频道/EPGDetails/OSS 兼容代码保留，旧 `epg.fetch_cron` 仅影响该兼容流程，新节目单更新周期以面板设置为准。

## 数据与项目结构

`IPTV_DATA_DIR` 可修改数据目录，默认 `./data`。`panel.json` 保存设置、运营商目录、扫描频道与手动覆盖，采用临时文件 + 原子替换提交。请备份该目录；仓库已忽略 `data/`、`.env` 和运行日志。

- `modules/panel`：配置存储、频道覆盖、扫描、识别、HTTP API 与嵌入式前端。
- `modules/spider`：独立的运营商客户端；不依赖全局认证状态或数据库。
- `initialize/panel.go`：新服务与原 MySQL/EPG 数据的适配。
- `modules/auth`、`model`：保留的 EPG/对象存储兼容能力。

```bash
go test -race ./...
go vet ./...
node --check modules/panel/web/app.js
```

自动测试使用本机模拟 HTTP 服务覆盖认证链、扫描/停止、持久化、访问控制和识别 API，不会扫描真实专网或调用付费模型。组播播放效果需在可访问转发服务的部署环境验收。

### 单播与回看

同步频道会保存运营商内部频道标识、RTSP 时移地址和回看能力。面板“账号与转发”可选择默认组播、RTSP 转 HTTP 单播或运营商 HTTP 直播；频道的“播放”按钮可选择方式和回看时间，也支持编辑单播地址及回看天数。RTSP 复用转发器 `/rtsp/` 路径，不附加组播 FCC。

HTTP 直播使用官方 `getChannelPlayUrl` 的 `liveUrl`；回看按 PHP 参考通过 `getPreCurNextProg` 和 `getTvodPlayUrl` 获取 HLS。每次播放重新认证取址，不缓存过期签名。HTTP 回看从所选开始时间播放，结束边界由运营商节目决定；RTSP 回看按中国标准时间拼接 `playseek` 起止时间。

M3U 带 `catchup="default"`、`catchup-days` 和 `{utc}/{utcend}` 回看模板。默认 7 天为参考 PHP 的配置值，实际可用时长与频道权限以运营商为准；0 关闭回看。播放列表中不具备所选单播方式的频道会被跳过。播放器需要能直接访问 HTTP 地址所在的运营商网络；网页中的“打开”不保证浏览器原生支持 HLS/MPEG-TS。

### TiviMate 与 Televizo 回看格式支持

系统原生兼容 **TiviMate** 与 **Televizo** 等主流 IPTV 播放器的回看（Catchup / Archive / Timeshift）功能：

- **M3U 头部与频道标准标签**：
  - `#EXTM3U` 头部默认输出 `catchup="default"` 与 `x-tvg-url`，播放器导入订阅时自动检测并开启回看支持，无需在播放器内繁琐调试。
  - `#EXTINF` 标签输出标准 `catchup="default"`、`catchup-days="7"`、`timeshift="7"` 与数字台号 `tvg-chno="51"`，便于电视遥控器数字键直接切台。
  - 回看源模板支持在 Web 面板**可视化配置**（预设包含 `&utc=${start}&lutc=${end}`、`&start={utc}&end={utcend}` 等，支持自定义）。默认使用 `catchup-source="…/api/play?id=…&mode=…&utc=${start}&lutc=${end}"`，为 Televizo 与国内安卓播放器提供开箱即用最佳体验。
- **全方位支持播放器回看参数**：
  - **TiviMate 变量**：原生支持 `{utc}`（UTC 开始秒级时间戳）与 `{utcend}`（UTC 结束秒级时间戳）。
  - **Televizo 变量**：兼容 `${start}` / `{start}`、`${end}` / `{end}`、`${timestamp}` 以及 `${duration}` / `dur` 参数（自动计算起止时间）。
  - **直播时移与正在播出节目“从头播放”**：对正在直播节目的“从头播放”请求，自动识别并容错未来结束时间，不再报错 HTTP 400，平滑无缝时移。
  - **多种时间格式解析**：兼容秒级时间戳、毫秒时间戳（13位）、北京时间紧凑格式（YYYYMMDDHHMMSS）、ISO8601 及 `playseek` 范围参数。
  - **HEAD 请求支持**：全面支持 HEAD 请求，解决 ExoPlayer / VLC 播放内核在流预探测时的 405 Method Not Allowed 报错。
- **播放器使用指引**：
  - **TiviMate**：添加播放列表，填入 M3U 订阅地址（点击面板「复制 M3U 链接」获取，如 `http://<IP>:8080/api/m3u8`）。EPG 自动关联，节目单中出现时钟回看标志，支持 7 天节目回放与时移。
  - **Televizo**：添加播放列表填入相同 M3U 地址，播放列表设置中的回看模式保持为“自动/默认”即可，支持时移播放与精确拖拽。

### 模型智能匹配台标

编辑频道时点击“模型智能匹配台标”，复用“智能识别”中保存的 API、key 和模型。将当前表单中的频道名称、分组和台标库目录发送给模型（纯文本，无需截图），返回目录内的候选、置信度和说明。置信度低于 80% 或无匹配时不提供可应用的台标。点击“使用此台标”后仍需保存频道，已有手动台标不会自动覆盖；可先填入频道识别建议再匹配。请求可能产生模型服务商费用。接口为受令牌保护的 `POST /api/panel/logos/match`，JSON 请求体为 `name`、`group`。

### EPG 面板和自动更新

EPG 页面支持开启自动同步、配置间隔（默认 6 小时）、历史天数（默认 7）和未来天数（默认 3），也可立即同步。同步进度、错误、存储模式和最近成功时间可见；局部失败保存成功频道并保留失败频道旧节目，写入失败不发布新数据。节目按北京时间浏览，提供正在播出和回看入口。XMLTV 使用当前频道 ID 和名称，ID 修改后仍通过运营商内部标识关联节目。M3U 默认附带本机 `/api/epg` 地址，已有 `epg.xml_url` 配置优先。

### 台标源配置

“智能识别”页面支持添加与管理多个台标源（支持 GitHub `/tree/分支/目录` 地址，及提供 `/api/list` 的台标库），不再区分国内国外源。多源按列表从上到下按优先级顺序匹配，优先匹配常规/亮色台标，深色台标自动作为次选后备。可在页面中自由调整源优先级、添加新源或移除失效源，并支持一键更新索引与本地保存快照。

### 一键探测与组内排序

- **一键探测分辨率**：在“频道管理”工具栏提供“一键探测”，支持后台批量探测所有启用流或未获取分辨率频道的视频规格（4K / 1080P / 720P），实时显示探测进度并更新各频道徽章。
- **组内频道排序**：在“IPTV配置”提供“组内频道排序”，可选择任意分组自定义该分组内部频道的绝对次序，支持拖拽移动、置顶/置底，或一键按台号顺序、字典序、分辨率优先重排。保存后立即应用于管理列表及 M3U/TXT/JSON 导出。
