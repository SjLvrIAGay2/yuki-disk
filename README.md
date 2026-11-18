# Yuki Disk Status

生产内部采集优先使用部署方固定的受认证直达地址；通过受限 SSH 隧道采集远端节点时，仍必须校验节点 TLS 证书并携带其独立监控凭据。不要让同机采集绕经公网 CDN 后再返回节点，否则公网往返延迟可能超过探测期限。此时页面反映内部服务健康，不等同于对每个公网接入路径的可用性保证；不因改用内部采集地址而开放匿名内部监控接口。

状态服务器使用原生 Node.js 24，可在 Linux、Windows 和 macOS 上独立运行，无需 WSL 或虚拟机。公开指标沿用相同白名单；平台不提供的指标显示未知。Windows 私有配置通过 ACL 限制访问，systemd 示例仅用于 Linux。

Yuki Disk Status 是独立服务状态页，公开展示服务健康摘要。它拥有独立进程、端口、配置与内部采集，不依赖网盘网页进程，不提供文件内容或文件写入接口。

匿名页面只展示整体可用性、聚合容量、CPU、内存、传输负载及采样时间，不展示服务名称、节点、拓扑或精确版本。web 管理页面显示服务详情与运行日志，公开 status 没有登录入口。web 配置固定 `statusService` 后，通过服务端只读代理读取本服务 `/api/details`；运行日志直接读取 control 的管理员接口。`/api/events` 保留私有接口兼容。管理员 web 会话逐次由 control 核验，普通账号和匿名请求拒绝。

## 构建

需要 Node.js 24，以及位于 `../../yukiUI` 的独立 UI 仓库。先按[项目获取步骤](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#获取与构建)准备锁定版本，再从 status 仓库执行：

```bash
npm ci --ignore-scripts
npm run build
```

依赖 `@yuki-ui/react: file:../../yukiUI`，保留 `.npmrc` 中的 `install-links=true`。构建核对 UI 编译产物摘要；更新编译包后需重新安装依赖。部署只需 `server.mjs`、`server/`、`dist/` 和 Node.js，无需 UI 源码、数据库或 npm 运行依赖目录。

## 配置

通过 `--config` 指定唯一 JSON 配置，修改后重启，无环境变量覆盖。参考 [server.example.json](server.example.json)；相对 `directory`、`cert`、`key` 按启动工作目录解析，生产建议使用绝对路径。

```json
{
  "host": "0.0.0.0",
  "port": 8443,
  "development": false,
  "publicOrigin": "https://status.disk.example.com:8443",
  "directory": "/opt/yukidisk/status/dist",
  "cert": "/etc/yukidisk/tls/fullchain.pem",
  "key": "/etc/yukidisk/tls/privkey.pem",
  "control": "https://api.disk.example.com:8443",
  "intervalMs": 10000,
  "timeoutMs": 3000,
  "services": [
    {"id":"control","name":"文件服务","kind":"control","url":"https://api.disk.example.com:8443/readyz"},
    {"id":"storage-1","name":"存储服务 1","kind":"storage","url":"https://hot1.disk.example.com:8443/readyz"},
    {"id":"storage-2","name":"存储服务 2","kind":"storage","url":"https://hot2.disk.example.com:8443/readyz"},
    {"id":"maintenance","name":"维护服务","kind":"maintenance","url":"https://maintenance.disk.example.com:8443/readyz"},
    {"id":"web","name":"网盘网页","kind":"web","url":"https://disk.example.com:8443/readyz"}
  ]
}
```

将示例替换为实际部署地址；同机服务使用不同监听端口。维护 URL 需要预先部署受控 HTTPS 代理，转到维护进程的本机健康端口。

| 字段 | 要求 |
|---|---|
| `host`、`port` | 监听地址，端口 1..65535；回环模式只能绑定回环 |
| `development` | 必填布尔，生产为 `false` |
| `publicOrigin` | 浏览器实际访问的精确 origin，无路径、尾斜线、查询或凭据 |
| `publicPath` | 默认空字符串；可配置 `/disk`、`/status` 等固定路径，不带尾斜线；修改需重启 |
| `directory` | 完整静态产物目录 |
| `cert`、`key` | 生产 HTTPS 证书及私钥，TLS 至少 1.2 |
| `control` | 用于登录、事件与集群元数据的控制面基础地址，可带 `/api` 等固定前缀，不带尾斜线 |
| `services` | 1..32 项，包含唯一 `id`、显示 `name`、`kind`、`url` |
| `kind` | `control`、`storage`、`maintenance`、`web` |
| `url` | 固定服务基础地址加 `/readyz`，如 `https://disk.example.com/api/readyz`；不含查询或片段；遥测取同一前缀下的 `/statusz` |
| `intervalMs` | 默认 10000，范围 1000..60000；每轮完成后等待此间隔 |
| `timeoutMs` | 默认 3000，范围 100..10000；单次探测期限 |
| `services[].monitorKey` | 与目标私有 monitor_key（web 为 monitorKey）匹配，至少 32 字节独立采集凭据，不使用管理/节点密钥 |
| `services[].hostGroup` | 同一物理/运行环境主机填写相同分组，CPU/内存只计一次；缺少分组则不发布这类聚合 |
| `services[].capacityGroup` | 存储容量去重分组；同一容量池填写相同值，独立容器/配额填写独立值；缺少则容量未知 |

生产 `publicOrigin`、`control` 和全部目标均为 HTTPS；`development=true` 仅允许回环 HTTP。重定向始终拒绝。合法配置、证书与静态产物缺失时拒绝启动；控制面或被监控服务离线不会阻止 status 启动，但相关健康、登录和记录会降级。

status 在服务端访问上游，不需要浏览器跨域访问控制面；control origins 须包含 status publicOrigin，以校验独立管理员登录的来源。使用组织 CA 时配置 Node.js 信任根，例如 `NODE_EXTRA_CA_CERTS`，不关闭证书校验。该变量用于运行时 TLS 信任，不覆盖应用 JSON 配置。

## 部署与启动

子路径部署时，反向代理须保留路径：例如 `publicOrigin=https://status.example.com`、`publicPath=/disk` 对应 `/disk/`、`/disk/assets/…` 和 `/disk/api/status`。`/disk` 自动重定向到 `/disk/`；无需代理改写 HTML、脚本或 CSP。浏览器只请求当前 origin 下的已配置路径。根路径 `/readyz`、`/healthz` 保留作进程探测，不开放私有监控数据。

同时发布 `status.example.com/disk` 和 `disk.example.com/status` 时，使用同一份编译产物启动两个独立实例，分别配置精确 `publicOrigin`、`publicPath`、不同监听端口及相同固定采集清单。两者各自缓存采样，每个实例均受采样间隔和超时约束；公开刷新不触发即时探测。仅在控制面配置中登记实际需要的两个 origin，不使用通配来源。web 的 `statusService` 填写所选实例的完整基础地址，包括路径。

兼容会话的续期 Cookie 使用该实例的 `publicPath + /api/session`，仍为 HttpOnly、SameSite=Strict，生产 Secure；控制面地址的 `/api` 前缀不应泄漏到错误的浏览器 Cookie 路径。公开页面不增加登录或管理入口。

创建 `yukidisk` 服务账户，安装文件到以下位置并保证对账户可读：

```text
/opt/yukidisk/status/
  server.mjs
  server/
  dist/
/etc/yukidisk/status.json
```

配置权限设为 `0640 root:yukidisk`，私钥限制为服务账户可读。前台启动：

```bash
node /opt/yukidisk/status/server.mjs --config /etc/yukidisk/status.json
```

长期运行可创建 `/etc/systemd/system/yukidisk-status.service`：

```ini
[Unit]
Description=Yuki Disk status page
After=network-online.target
Wants=network-online.target

[Service]
User=yukidisk
Group=yukidisk
WorkingDirectory=/opt/yukidisk/status
ExecStart=/usr/bin/node /opt/yukidisk/status/server.mjs --config /etc/yukidisk/status.json
Restart=on-failure
RestartSec=3
TimeoutStopSec=10
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
UMask=0077

[Install]
WantedBy=multi-user.target
```

按 Node.js 安装位置调整 `ExecStart`，证书不放在 home 中。启用并检查：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now yukidisk-status
curl --fail https://status.disk.example.com:8443/readyz
curl --fail https://status.disk.example.com:8443/api/status
sudo journalctl -u yukidisk-status -n 50 --no-pager
```

本机回环运行可使用 `npm start`，默认 `http://127.0.0.1:5176`，需先调整示例中的目标地址；示例网盘目标为 5175，而 web 默认示例端口为 5173，必须与实际部署一致。

## 采集与网络

节点 `ready` 读取最近完成的对账结果，不等待文件传输持有的存储锁；未完成首次对账、对账失败或成功结果超过 30 秒时为 false。status 尊重该结果，仍独立执行采样超时和缓存过期判断，不将成功的 HTTP 响应自动等同于服务正常。

每轮优先读取固定目标的 `/statusz`。响应需包含 `version=1`、与 kind 对应的服务类型和布尔 `ready`；不满足时回退同 origin 的 `/readyz`。storage 的就绪正文为 `ready:true`，其他服务为 `status:"ready"`，同时要求 HTTP 成功。正常每目标每轮一次请求，回退最多两次；单次各自受 `timeoutMs` 限制。

`/statusz` 要求独立 X-Yuki-Monitor，status 使用每个目标的 monitorKey，未配置或错误时只可能取得基础就绪状态，资源指标显示未知。还应通过防火墙或反向代理限定采集网络。维护原 HTTP 端口保持本机绑定，远程采集通过受控 HTTPS 代理。

生产反向代理应保留准确 `publicOrigin`。服务不信任 `X-Forwarded-For`，代理后可能将用户归入同一来源限流，应按实际部署配置入口限流。多 status 实例通过控制面的 PostgreSQL 共享会话，不要求固定路由；各实例须使用同一 publicOrigin 和控制面身份域。

## 资源详情与维护记录

内部详情接口提供进程 CPU、内存、系统资源，以及节点存储方式、配置容量、已存内容、可用空间、在途预留和收发速率，供 web 管理页面使用。控制面提供冷热角色、副本与任务统计；维护进程提供等待、执行或失联，以及本次启动已确认完成和失败的任务数。公开页面不请求或呈现这些数据。

CPU 与 I/O 速率由同次启动的相邻累计值计算，首样本、重启或采集失败时不显示为零。进程 CPU 以单核 100% 计，可超过 100%；系统 CPU 以全部逻辑 CPU 计。Linux 可提供实际磁盘 I/O，系统资源是当前运行环境可见值，不换算容器 cgroup 配额。

存储容量属于节点文件系统或容器，同卷节点不能简单累加；目录配额不包含全部元数据，容器有效载荷与占用页数不同。带宽预算只作调度权重，不是限速。健康失败或数据过期时隐藏旧详情，恢复后重新采集。

兼容维护事件接口只读，按增量分页读取，每页最多 200 条；上游事件按控制面的保留规则清理，不等于运行日志。status 不保存历史资源或可用率数据库。就绪探测不能证明全部文件完整或所有浏览器资源可加载。

## HTTP 接口与会话

| 路由 | 权限与用途 |
|---|---|
| `GET /api/status` | 匿名聚合，HTTP 200；state 为 ready/degraded/stale/unavailable，不接受查询参数 |
| `POST /api/session` | `{username,password}` 管理员登录，返回 status 专用 `token` 和 `expiresAt` |
| `POST /api/session/refresh` | HttpOnly Cookie 续期，返回新的短期访问令牌 |
| `DELETE /api/session` | Cookie 撤销当前 PG 会话并清除 Cookie，幂等注销 |
| `GET /api/events?after=0` | Bearer 管理员 web 或兼容 status 令牌，返回脱敏维护记录及 `nextAfter`、`more` |
| `GET /api/details` | Bearer 管理员 web 或兼容 status 令牌，不接受查询参数，返回资源与元数据可用状态 |
| `GET /healthz`、`GET /readyz` | status 自身就绪，不因被监控服务故障而失败 |

错误为 `{code}`，状态码包括 400、401、403、404、405、413、415、429、503。客户端不能指定代理 URL，不能通过 status 访问文件内容。

内部读取接受 web 管理员 `session` 令牌，并兼容独立的 `client:status` PG 只读会话。两者均在每次读取时由 control 核验签名、账号角色、会话和撤销；语法解析不是身份依据。兼容访问令牌最长 5 分钟，其 `status-session` 用途在控制面只允许读取状态、维护事件和当前身份，不能管理账号、配置、应用或读取文件。兼容长期凭据为独立 `yuki_status_refresh` Cookie，HttpOnly、生产 Secure、SameSite=Strict、Path=/api/session、无 Domain；公开页面不创建或恢复该会话。status 只转发该 Cookie 到控制面的会话入口，不保存凭据或会话副本。绝对期限 30 天、空闲期限 7 天、轮换重试窗口 30 秒，与控制面一致。

会话写入和续期要求准确的 publicOrigin 与 `X-Yuki-CSRF: 1`；控制面的 origins 必须包含 status 的 publicOrigin。浏览器刷新或重新打开后先续期再显示内部内容，多标签页通过浏览器锁合并续期、广播退出；匿名状态采集和展示不依赖登录恢复。退出必须完成 PG 撤销，服务重启不会使旧会话丢失或复活。每次内部读取继续核对 PG 管理员状态；禁用、重置、角色改变后拒绝，控制面无法验证时也不返回缓存内部详情。登录最多 4 路并发、单 IP 每分钟 10 次，事件最多 16 路并发，详情最多 4 路并发。系统管理操作使用 web 的管理员页面。

SIGINT/SIGTERM 停止接收请求，取消采集与上游请求，5 秒后强制关闭剩余连接；PG 会话保留至退出、撤销或到期。JSON 日志只包含请求状态与耗时，不记录密码、令牌或事件详情。

## 公开指标白名单

只由 server/public-summary.mjs 从采集缓存投影下列字段。公开接口不得展开上游对象；新增字段必须评审来源与暴露风险。所有指标按 intervalMs 更新，HTTP 刷新只读缓存，不触发采集。采样超过 intervalMs + 2×timeoutMs + 2 秒视为过期，失败/缺失值为 null，不冒充零或正常。

| 字段 | 来源、聚合与精度 | 暴露风险 |
|---|---|---|
| state、checkedAt、stale、intervalMs | 固定目标就绪结果归并为整体状态；最近采集完成时间与配置采样周期 | 暴露整体故障和更新时间，不暴露目标 |
| capacity_bytes、available_bytes | 节点 storage.total_bytes/available_bytes；按 capacityGroup 去重求和，四舍五入到 64 MiB | 暴露粗粒度部署规模与使用变化；不含路径、后端或副本位置 |
| storage_used_percent | 聚合原始容量计算百分比，四舍五入整数 | 暴露整体空间使用趋势 |
| cpu_percent | hostGroup 去重、按逻辑核数加权的系统 CPU，整数百分比 | 暴露整体计算负载，不含主机名、核数或进程列表 |
| memory_used_percent | hostGroup 去重后总内存与可用内存求和，整数百分比 | 暴露整体使用比例，不含进程内存或精确容量 |
| receive_bytes_per_second、send_bytes_per_second | 同次启动的节点 received_bytes/sent_bytes 计数差除以采样间隔，再跨节点求和，四舍五入到 64 KiB/s | 反映副本传输在内的汇总活动，不含用户、文件、任务或方向目的地；不把历史峰值带宽当作当前负载 |

容量是节点提供的可用池口径，不代表用户可上传的独占逻辑容量；副本和预留仍影响调度。管理员必须正确填写去重分组，不要把同卷容量重复计算。首个速率样本、未配置存储目标或采集失败时显示未知。

每个 status 进程按 socket 来源 IP 每分钟限制 120 次公开读取，最多 2048 个来源计数；入口可进一步限流。多实例的公开限流分别计数，反向代理不能直接信任来访 X-Forwarded-For。后台每目标每轮最多两次有界探测，采集故障不阻塞网盘接口。匿名读取不能指定 URL、执行命令或触发维护。精确版本、地址、节点标识、配置、令牌、用户/文件、日志与原始错误均不在白名单。

## 配置分类

| 配置 | 类别 | 原因 |
|---|---|---|
| host、port、development、publicOrigin、cert、key、directory | 必须重启 | 监听、TLS、来源校验与静态索引 |
| control、services 全部字段及 monitorKey/hostGroup/capacityGroup | 必须重启 | 采集目标、信任与聚合口径不可由匿名请求改变 |
| intervalMs、timeoutMs | 必须重启 | 后台采样器启动时固定，已有采样不被部分更新 |
| 浏览器刷新 | 不适用配置热修改 | 只读取后台缓存，无配置写入接口 |

status 不提供热修改或任务执行入口。严格 JSON 为唯一应用配置；系统 TLS 信任环境变量只在进程启动时读取。秘密只保存在服务端私有配置。

## 运行日志

公开页不展示服务详情、维护记录或运行日志入口。运行日志只在 web 管理员页通过控制面鉴权读取；原私有详情/事件 API 保持鉴权兼容，公开白名单不扩大。

私有服务器配置 `logKey` 与 control.log_keys.status 配套，至少 32 字节且独立于监控凭据，只可签名写 `/internal/v1/logs`，不允许其他内部操作。未配置时该模块不集中上报；该字段与采集目标等启动配置一样必须重启，不能通过公开 API 查询或修改。状态服务独立维护最多 1000 条内存缓冲，默认 10 秒批量上传启停、请求失败、监测异常/恢复转换，不记录每次成功探测或公开读取，不上传目标标识、URL、异常或配置。PG 策略响应热更新日志周期/级别，下一批报告确认版本；保留默认普通 1000 条、警告/错误 30 天。上报失败退避至 60 秒，退出限时排空，崩溃可能丢失未送达事件，不能阻塞采样和匿名读取。
