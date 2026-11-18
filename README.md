# Yuki Disk Control

维护扫描批量读取当前事务的副本位置及任务状态，避免为每个稳定块重复往返数据库。按块和代际隔离，执行中任务及最近五分钟失败复制仍阻止重复规划；全局序列化、逐项预留、提交和物理执行条件保持一致。

运行日志接收固定的对账错误分类代码（见顶层“接口与一致性约定”），拒绝任意错误文本。新旧节点日志兼容，保留、幂等上传及管理员读取权限不变；日志生产者升级前先升级本服务的白名单。

容量不足时，上传计划在保存点中完整回滚，只在原鉴权事务内提交最多 256 个安全回收块的调度，响应仍为 `503 insufficient_capacity`、`Retry-After: 2`，客户端保留同一幂等键重试。维护扫描也会在新鲜可写节点无法容纳标准块副本时触发。仅提前处理无版本引用、无有效未提交上传引用、无未确认空间预留及无复制任务的零引用块；已提交上传的临时保留时间不会拖延永久删除后的紧急回收。恢复期限不缩短，挂载源文件不删除，空间以物理回收后新的签名资源报告为准。任务 `capacity_gc` 优先领取，继续使用原删除许可、代际和重试协议；成功完成后清除对应节点旧代际的预留，不影响新代际。无需 schema 升级。

节点配置查询保留旧进程报告，旧报告超过 20 秒且有较新的同节点报告时返回 `state=superseded`；当前最新报告过期仍返回 `stale`，同时新鲜的失败报告不会被隐藏。web 按服务汇总应用结果，不展示内部并发版本。其他多实例服务的失联报告仍参加汇总，不能由单个已应用报告掩盖。

Yuki Disk 控制面负责登录、目录与版本、回收站、块去重、副本分配、公开访问和维护任务。所有元数据与任务状态存储在 PostgreSQL；文件内容直接在客户端与节点之间传输。

## 构建与启动

控制面原生运行于 Linux amd64、Windows amd64 或 macOS arm64，使用真实 PostgreSQL 16，构建使用 Go 1.27.1。数据库可以独立部署，不要求 Windows/macOS 安装 Linux。以下命令在本仓库目录执行：

```bash
mkdir -p bin
go build -trimpath -o bin/control .
bin/control -config /etc/yukidisk/control.json
```

Windows 使用 `go build -trimpath -o bin/control.exe .`，然后执行 `bin\control.exe -config C:\ProgramData\yukiDisk\control.json`。macOS 使用相同 Go 构建命令和本机绝对配置路径。Unix 文件权限与 Windows ACL 都必须限制私有配置读取范围；监听、TLS、数据库及会话协议不随操作系统变化。下文 systemd 示例仅用于 Linux。

使用 [config.example.json](config.example.json) 查看完整字段，示例仅适用于回环环境。生产建议按[完整部署说明](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#配置数据库与集群)生成配套密钥，再将配置安装为 `0640 root:yukidisk`。数据库角色须拥有项目 schema 的迁移权限，远程数据库连接使用 `sslmode=verify-full`。

首次启动在数据库锁与事务内执行迁移；发现比程序更新的迁移版本时拒绝启动。多实例必须连接同一数据库并使用相同节点、挂载、窗口和密钥配置。

## 配置

启动配置通过 `-config` 指定，严格 JSON，拒绝未知字段和尾随内容，无环境变量覆盖。运行配置的覆盖关系与热修改范围见下文。

| 字段 | 含义 |
|---|---|
| `listen` | 监听地址，默认 `127.0.0.1:8080` |
| `database` | PostgreSQL DSN，必填 |
| `development` | 生产设为 `false`；回环 HTTP 仅可用于 `true` |
| `tls_cert` / `tls_key` | 生产必填的证书及私钥路径 |
| `password` | 仅首次迁移初始化指定普通用户的旧凭据，之后 PostgreSQL 账号为权威 |
| `bootstrap_admin` | 首次初始化管理员的 `{username,password}`，必须由部署者提供；数据库已初始化后不覆盖凭据 |
| `legacy_username` | 明确接收存量文件、目录、上传和旧分享规则的普通账号；已有数据时必填 |
| `monitor_key` | 独立只读诊断凭据，至少 32 字节；未设置时诊断端点关闭 |
| `session_key` | 用户会话签名密钥 |
| `capability_key` | 节点数据能力签名密钥，与节点配置一致 |
| `maintenance_key` | 维护身份密钥，与维护及节点配置一致 |
| `nodes` | 节点 ID、origin、角色、独立密钥及调度参数 |
| `origins` | 允许的网页及播放器精确 origin |
| `replicas` | 完整副本数量，默认 2，允许显式设置为 1，且不大于节点数量；修改需重启所有 control 实例 |
| `trash_seconds` | 回收站恢复期限，默认 604800 秒（7 天） |
| `gc_seconds` | 块回收延迟，默认 86400 秒（1 天） |
| `upload_seconds` | 上传保留期，默认 900 秒，最低 120 秒 |
| `cold_seconds` | 冷数据迁移阈值，默认 604800 秒 |
| `mounts` | 只读服务器目录映射，默认空数组 |
| `p2p` | 可选 broker origin、issuer identity 和独立 key；可配置内部 `issuer_url`、绝对 PEM 路径 `issuer_ca_file` |

密码及各类密钥至少 32 字节，彼此不同；每个节点也使用独立密钥。生产使用随机生成的值。所有 URL 都是无路径、尾斜线、查询和凭据的 origin。

`nodes` 中每项包含 `id`、`url`、`tier`（`hot` 或 `cold`）、`key`；可选 `read_bandwidth_bps`、`write_bandwidth_bps`、`reserve_bytes` 和 `p2p_service`。节点不能自行注册任意 URL。新增节点需同步更新控制面、维护、客户端节点白名单和各节点复制来源白名单。

## 自动分配与迁移

读写带宽单位为字节/秒，0 表示自动估计，正数仅作调度预算，不限速。节点签名资源报告提供实际收发速率、并发数、可用空间和分配单位。空间安全余量为 `max(reserve_bytes, 总容量的 5%)`。

资源接收超过 15 秒或采样超过 20 秒即失效，心跳超过 30 秒视为离线。新写入只分配到新鲜、可写、足够容量的节点；完整副本目标无法满足时返回 `503 insufficient_capacity`。同卷目录节点共享物理预留，容器按已预分配内部空间计算。没有新鲜样本的已有在线副本仍可作为较低优先级读取来源。

空间预留按实际分配单位取整，绑定块、节点和代际，默认 120 秒；重试复用预约。下载目标按负载排序。热度按用户获取块下载授权增长，半衰期 5 分钟，达到 8 视为热块；清单查询不增加热度。副本不足优先修复，其次根据冷阈值、容量和带宽进行迁移。

低于 10% 空闲时可向另一分配卷且空闲至少 20% 的节点迁移；热块可向读取预算至少 2 倍且评分改善 1.5 倍的 hot 节点迁移。容量和带宽重平衡间隔至少 120 秒。所有迁移先复制确认，再删除多余副本。

单副本按块选择一个有新鲜签名资源、满足空间预留及安全水位的节点，同一文件的不同块可以位于不同节点。提交仍要求当前代际完整副本与上传归属证明，不能只凭 HEAD。单副本没有节点故障冗余，唯一副本丢失后无法由系统恢复。副本数是启动配置，不属于运行时热修改；多实例必须统一配置并协调重启，不能混用不同副本策略。降低副本数后，维护流程会逐步回收多余副本，保留完整副本计数、执行租约及代际保护；调回更高副本数不会立即恢复冗余，必须等待复制完成并验证。变更前保留受保护的配置和程序备份，不直接修改 replica 元数据或容器格式。

上传规划在同一个 PG 串行化事务内复用资源与预留视图，每次预留同步扣减节点配额和共享卷可用量，避免逐块重复扫描全部预留。视图不跨事务缓存；遇到块复活或代际变化时重新读取权威状态。整份规划仍原子提交或回滚，不以拆分事务缩短锁等待。

## 只读目录与公开访问

节点先配置 `exports`，管理员通过网页建立来源与用户目录授权，见下方“账号删除与目录挂载管理”。以下控制面配置仅用于旧部署的首次导入兼容：

```json
{
  "mounts": [
    {"id":"server-media","path":"服务器文件","node":"n1","export":"media"}
  ]
}
```

将该字段合并到完整配置。节点 `n1` 必须已登记，映射路径不能被文件、文件夹或在途上传占用，映射之间不能嵌套。外部目录只读，不生成托管块副本或历史版本。该 ID 导入后不再随 JSON 改动或重启修改；移除、重新选择来源及分配用户均在管理员页面进行，移除不删除服务器文件。

旧路径可见性数据保留为归属迁移清单；`/public/v1/browse` 与 `/public/v1/content` 返回 `410 legacy_share_expired`，不会将旧路径授权套用到多用户同名路径。新的用户分享与机器 API 分享见下文。存量挂载保留在指定存量普通用户下，管理员新增授权时显式选择接收账号，不自动对新账号开放。

下载要求节点 HTTPS 地址可达；控制面不代理文件字节。状态查询及权限撤销都以数据库为准，数据库不可用时拒绝新增授权。

## HTTP 接口

除登录、匿名内容与健康接口外，用户接口要求 `Authorization: Bearer <session>`。

| 路由 | 用途 |
|---|---|
| `POST /v1/session` | `{username,password,client}`，`client` 为 `web`、`cli` 或内部只读 `status`；访问令牌 300 秒 |
| `GET /v1/contract` | 块大小、哈希算法、副本要求 |
| `GET /v1/browse`、`POST /v1/folders` | 目录、搜索、分页与建目录 |
| `POST /v1/operations` | rename/move/copy/trash/restore，最多 100 个源路径 |
| `GET /v1/files`、`GET /v1/versions` | 文件和版本列表 |
| `POST /v1/uploads`、`POST /v1/uploads/{id}/commit` | 上传计划与完整副本提交 |
| `GET /v1/download/{id}` | 下载计划；`manifest=true` 只取清单 |
| `GET /v1/download/{id}/blocks/{hash}` | 为该版本的块重新领取一次性能力 |
| `POST /v1/trash`、`POST /v1/restore` | 按路径删除或恢复 |
| `POST /v1/trash/actions` | `{operation,items:[{path,trash_id}]}`，restore/purge |
| `POST /v1/versions/actions` | `{operation,items:[{id,trash_id?}]}`，trash/restore/purge |
| `GET /v1/mounts`、`GET/POST /v1/visibility` | 只读映射与可见性 |
| `GET /v1/status`、`GET /v1/blocks/{hash}` | 集群与块健康、资源和分流状态 |
| `GET /v1/events`、`GET /v1/tasks` | 维护记录与任务 |
| `POST /v1/p2p/{node}` | 空 JSON、会话及白名单 Origin 换取短期连接身份 |

列表每页最多 200 项；文件与目录使用 cursor，事件与任务使用 after。上传、树操作、回收站及版本批次需要稳定幂等键。版本和回收站批次的 restore/purge 必须绑定当前 `trash_id`，身份已变返回 409，过期恢复返回 410；批次最多 100 项，整批事务执行。网页清空回收站是多批操作，不承诺跨批全局原子性。

内部 `/internal/v1/report`、`inventory-page`、`resources`、`claim`、`permit`、`complete` 和 `content` 使用服务身份签名；具体认证与一致性约定见[顶层协议说明](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#接口与一致性约定)。

`POST /internal/v1/upload-receipt` 仅接受节点签名和 `{upload_id,hash,size,generation}` 元数据，在同一 PG 事务中登记增量副本与上传证明。清单、期限、大小或代际校验失败会回滚整个回执；重试沿用原上传身份，提交仍需完整副本。旧的 report/upload-proof 接口继续兼容。先升级控制面，再升级节点。

P2P 申请不在数据库事务中等待外部网络：事务内验证普通用户会话，释放锁后申请连接身份，再在新事务中复核原会话有效性与角色，成功提交后才返回凭据。管理员不能调用该用户接口。`p2p.issuer_url` 可指定固定内部 HTTPS 地址，`issuer_ca_file` 可配置专用 CA；仍验证证书及主机名、拒绝重定向。浏览器只收到公开 `broker` 地址。私有 CA 不影响其他 HTTP 客户端；文件须为不超过 1 MiB 的有效 PEM，配置需重启。内部地址与私有 CA 均不配置时兼容原公开 broker。

## 服务管理

使用 [systemd 配置](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/deploy/yukidisk-control.service) 将二进制安装到 `/opt/yukidisk/control`，配置安装到 `/etc/yukidisk/control.json`，再执行：

```bash
sudo install -m 0644 ../deploy/yukidisk-control.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now yukidisk-control
sudo journalctl -u yukidisk-control -n 50 --no-pager
```

证书及配置对服务账户可读，业务监听使用非特权端口或适当配置权限。`/healthz` 检查进程，`/readyz` 检查 PostgreSQL。`/metrics` 与 `/statusz` 需要 `X-Yuki-Monitor` 独立凭据；部署还应限定监控网络。`/v1/status`、`/v1/events`、`/v1/tasks` 和块诊断仅允许管理员。运行日志缓冲后集中入库；警告/错误另有脱敏 stderr 后备。SIGTERM 最多排空 10 秒。

升级前备份 PostgreSQL、节点数据及密钥。迁移只向前；回滚必须使用匹配的备份与程序。完整集群启动、备份和恢复见[项目 README](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#日常运维与恢复)。

## 多用户初始化与归属迁移

PostgreSQL 16 是身份、角色、账号状态、资源归属和撤销的权威。数据库服务角色不得为 superuser 或 BYPASSRLS。所有普通用户操作使用事务内 owner_id 范围与强制行级策略；块与维护任务仍是全局资源，通过既有引用、代际和互斥维护。管理员仅访问管理接口，不能用其令牌访问网盘接口或通过 CLI 登录；管理权限不包含浏览用户文件。

首次部署必须在私有配置显式指定 `bootstrap_admin:{username,password}`，无固定生产密码。普通用户由管理员创建，不开放注册；账号删除采用 30 天保留期。原单用户数据升级须先停止旧版写入、备份 PG 和节点根，然后指定 `legacy_username` 为存量数据的唯一普通用户，原 `password` 仅作为此用户的初始凭据。执行：

```bash
control -config /etc/yukidisk/control.json -migrate-only
```

迁移到 schema 17，在同一 PG 事务内保留路径、目录、版本、上传任务、树操作幂等记录、挂载与原 visibility，并为单用户数据设置固定 owner_id `legacy`；该身份绑定显式指定的用户名。schema 15 将现有挂载保留为同用户授权并建立独立来源，增加可恢复账号删除及永久 API 期限语义。已有有限分享不延长；老的未绑定用户挂载能力失效，客户端重新读取目录即可取得新能力。缺少归属或管理员凭据时整体回滚。全局块、位置及维护任务不改归属。旧公开链接明确失效为 410，不自动转换为新分享。部署者检查迁移计数后，用该普通用户核对文件、回收站及未完成上传，再升级全部协议消费者。不得把原始数据库交给新管理员充当个人网盘。

已初始化数据库不会再次应用 bootstrap 密码；完成初始化后删除私有配置中的 `bootstrap_admin`。密码使用随机盐 PBKDF2-HMAC-SHA256、600000 次迭代存储。`GET/POST /admin/v1/users` 列表和创建普通用户；`POST /admin/v1/users/{id}` 使用 `version` 乐观锁更新 `disabled`、`role` 或 `password`。这些安全状态修改增加 auth_epoch 并撤销该账号全部会话，恢复账号不会恢复旧会话。最后一个可用管理员不能被禁用或降级。角色修改不会转移或删除原资源。

## 浏览器与 CLI 会话

`POST /v1/session {username,password,client:"web"}` 要求精确匹配配置的 Origin 和 `X-Yuki-CSRF: 1`，返回 300 秒访问令牌及 `{user:{id,username,role}}`。长期随机凭据存于 `yuki_refresh` Cookie：HttpOnly、生产 Secure、SameSite=Strict、Path=/v1/session、无 Domain。部署 web/control 必须使用同站 HTTPS；跨站域名需由反向代理统一站点。CORS 仅允许列举的 origin 与凭据，不支持通配符。访问令牌只放页面内存，不放 URL 或 localStorage。

会话绝对期限 30 天，空闲期限 7 天。`POST /v1/session/refresh` 同样要求 Origin/CSRF，并在 PG 事务中轮换摘要；前一代凭据有固定 30 秒重试窗口，窗口内返回相同后继，不滑动延长窗口。服务重启不丢失状态。每次受保护请求核对 PG 账号及会话。`GET /v1/session/me` 恢复身份；`DELETE /v1/session` 使用 Cookie 撤销当前服务端会话并过期 Cookie。禁用、重置凭据和角色变化即时阻断旧访问及续期令牌。

CLI 使用 `client:"cli"`，仅普通用户允许，无浏览器 Cookie；短期令牌过期后用私有配置凭据重新登录，原操作保持幂等身份。省略 username 的旧 CLI 配置只映射显式 legacy_username，不能选择其他账号。旧无 PG 会话的签名令牌均失效。

status 使用管理员专用 `client:"status"`、独立 `yuki_status_refresh` Cookie 和 `status-session` 访问用途。控制面仅允许该访问令牌 GET `/v1/status`、`/v1/events`、`/v1/session/me`，不授予系统写入或网盘权限。续期和退出须传 `{client:"status"}` 并匹配该 Cookie 与 PG client；默认 web 会话不能消费 status Cookie。status 将 Cookie Path 改为自身 `/api/session` 后代理同一持久会话协议，期限、轮换及撤销规则不变。

## 两类文件分享

用户分享：`POST /v2/shares {version_id,kind:"user",expires_in,password?}` 与 Idempotency-Key 创建固定文件版本授权，期限 60 秒至 30 天；`GET /v2/shares` 列出自己的分享，`POST /v2/shares/{id}/revoke` 撤销。链接为 web `/share/{id}#access_key`，秘密不进入服务器 URL。接收者以 access_key 作为 Bearer 调用 `POST /public/v2/shares/{id}/access {password}`，换取用途独立的五分钟 share-download 令牌。随后 GET `/public/v2/shares/{id}/download` 读取清单，GET `/public/v2/shares/{id}/blocks/{hash}` 取得短期节点授权。只返回文件名、大小和所需块，不返回其他目录、所有者身份或内部错误。缺少访问凭据为 401，过期/撤销为 410，资源不可用为 404。

API 分享：普通用户以 `kind:"api"` 创建单版本授权并提供授权编号；管理员用 `POST /admin/v1/applications {grant_id,name,expires_in}` 和 Idempotency-Key 创建独立应用凭据。响应包含应用 id 与 credential，凭据只交付外部系统秘密配置。GET 同路径仅列配置摘要，POST `/{id}/revoke` 撤销。管理员不能据此列出普通用户文件或自行选择未授权资源。

机器接口版本为 `/api/v1`，仅支持下载：Bearer application credential 调用 GET `/api/v1/files/{app_id}` 和 `/api/v1/files/{app_id}/blocks/{hash}`，再凭返回的节点能力令牌直接 GET 节点块并验证完整长度与 SHA-256。控制面不收发块内容，不提供上传或泛化远程网盘接口。应用凭据、会话和用户分享令牌不可互换；创建接口相同幂等键和参数返回同一资源，参数变化拒绝 409。每个分享/应用每分钟最多 600 次下载授权，登录和分享访问入口每来源每分钟最多 20 次；PG 保存限流与审计。多实例共享这些限制。

上述 600 次和 20 次是初始默认上限；管理员可按“配置分类与运行版本”调整，实际请求遵循当前实例已应用的 PG 配置。

分享固定版本：移动不影响，新增版本不替换被分享内容；版本或路径进入回收站、永久删除时永久撤销，恢复不恢复分享。所有者被禁用或改为管理员时拒绝访问，恢复普通用户角色和可用账号后，未撤销且未过期授权可继续使用。每次下载清单和块授权检查最新状态。已签发节点能力有效期最多 60 秒，已开始的单块请求可完成；撤销不能收回接收者已下载字节，后续授权立即停止。

跨用户块去重不把“知道哈希”视为拥有内容。没有本用户既有版本引用或有效同代际上传证明时，prepare 返回 skip:false；客户端向资源报告新鲜的现有副本 PUT 完整数据，节点在原互斥、代际及完整哈希验证之后发送签名 upload-proof。PG 核验上传归属、清单、当前副本及代际后记录证明；commit 同时要求证明与完整副本条件。既有布局、预分配与持久描述符提交规则不变。

## 配置分类与运行版本

| 配置 | 类别 | 原因与影响 |
|---|---|---|
| trash_seconds、upload_seconds | 可热修改，仅新删除/新上传任务 | 保留既有期限；范围分别 60..7776000、60..86400 秒 |
| user_share_seconds、api_share_seconds | 可热修改，仅新建表单默认值 | 用户分享 60..2592000 秒；API 为 0（永久）或 60..3153600000 秒；不延长已有授权 |
| login_attempts、download_authorizations | 可热修改，实例应用后下一请求 | 分别 1..120 次/来源/分钟、1..6000 次/分享或应用/分钟；PG 共享计数不清空，非带宽限制 |
| listen、development、tls_cert、tls_key、origins | 必须重启 | 监听、TLS、跨域信任绑定进程 |
| database、password、bootstrap_admin、legacy_username、session_key、capability_key、maintenance_key、monitor_key | 必须重启或专门凭据流程 | 数据库、首次归属、安全材料不可任意在线替换；初始化项之后不重置账号 |
| nodes 的 read_bandwidth_bps、write_bandwidth_bps、reserve_bytes | 可热修改，仅影响新调度 | 首次启动种子，此后按节点 PG 配置优先；不重写已有任务和事务预留 |
| nodes 其余字段、replicas、gc_seconds、cold_seconds、p2p 全部字段、log_keys | 必须重启 | 节点身份、拓扑、安全凭据、放置与回收契约须一致部署 |
| mounts 启动兼容种子 | 仅首次导入该 ID | 已导入项由 PG 管理，不因重启或旧配置恢复已撤销授权；新部署使用管理接口 |

启动 JSON 仅在 PG 尚无 runtime_configuration 时提供上传与回收初值；新增分享默认值均为 86400 秒、验证次数 20、下载授权次数 600。之后 PG 持久配置优先，重启不会被 JSON 覆盖；没有环境变量运行时覆盖。管理员 GET/POST `/admin/v1/configuration`，提交 `{version,settings:{trash_seconds,upload_seconds,user_share_seconds,api_share_seconds,login_attempts,download_authorizations}}`。兼容旧客户端省略字段时从同一事务的 PG 目标值继承；完整校验、CAS 和目标更新同事务完成，非法或冲突不改变现值。普通用户 GET `/v1/preferences` 只能读取两种分享默认值。

## 账号删除与目录挂载管理

管理员 POST `/admin/v1/users/{id}/deletion`，正文 `{version,operation:"delete",username}` 必须提供当前版本及完整账号名称。立即禁用、增加鉴权代际并撤销所有会话与该用户分享，30 天内用 `{version,operation:"restore"}` 恢复；不恢复旧会话与分享。隔离期间暂停文件/版本回收倒计时，恢复后保留原剩余期限；未完成上传按原截止时间失效。到期维护事务复用版本引用释放，清除个人目录、版本、上传、幂等记录、分享、应用及目录授权，保留匿名账号墓碑以维持审计关联。全局维护任务继续由既有代际、互斥及对账机制驱动，其他用户共享引用不受影响。删除、恢复、到期清理共享事务锁；过期即不可恢复，失败回滚并在后续维护重试。最后一个可用管理员不能禁用、降级或删除。

节点先由部署者配置 `exports` 白名单并重启。管理员 GET `/admin/v1/exports?node=<id>` 列出导出 ID，加 `export`、`path`、`cursor` 浏览目录元数据；不会返回宿主机路径或文件内容。POST `/admin/v1/mounts` 正文 `{name,node,export,source_path}` 创建来源，必需 Idempotency-Key，相同键与参数安全重试；幂等结果沿用 7 天保留。GET 同路径返回来源、版本及授权。POST `/{id}` 使用 `{version,operation:"grant",owner_id,path}` 授予可用普通用户；目标必须为空且无冲突上传。`operation:"revoke",grant_id` 取消一项授权，`operation:"remove"` 删除来源及全部授权。修改均版本 CAS、审计及原子提交；取消授权不删除虚拟父目录或外部文件。挂载只读且按用户检查，管理员没有普通用户网盘文件读取权限。

API 来源与应用均支持 `permanent:true,expires_in:0`，响应 `expires_at:null`。有限期为 60 秒至 36500 天；永久应用必须引用永久来源授权，否则 409。有限应用到期时间取请求期限和来源期限较早者。永久仅表示没有时间截止，不豁免撤销、账号状态或源版本检查。用户公开分享不支持永久。

控制服务各进程每 2 秒拉取系统默认值，原子替换请求快照，分别报告目标版本、生效版本和应用状态；失败保留旧配置，10 秒未报告标记 stale。页面不能把部分成功显示为全部成功，这类可能多实例的服务查询保留一天内的报告，重启前无法确认退出的实例仍显示报告过期。节点独占根的旧进程报告使用前述 superseded 规则。恢复旧数值也需读取最新版本后提交新版本。配置更新不改变已有连接、上传截止与用户恢复期限；块 GC 宽限期仅在前述容量压力安全条件满足时可以提前。秘密不通过管理查询、审计或页面回显。

## 挂载范围授权

支持范围读取的节点升级后，普通用户挂载下载清单的 source 包含 node、p2p 和 ranges 标记。`POST /v1/download/{id}/ranges` 接收 `{size,offsets}`，最多六个按 4 MiB 对齐的不同起点；返回相同顺序的 `{offset,size,revision,range,target}` 数组。ID 必须是当前用户的签名 mount-version，PG 目录授权和账号仍有效。管理员不能调用该普通用户接口。总大小在节点与真实文件再次核对，控制面不探测文件内容。

范围能力与 file-read 分开，绑定精确范围及总大小，60 秒有效；内部 content 回查同时验证节点、所有者角色/状态、目录授权和用途。撤销阻止新回查，已准入的在途段可以完成。客户端重试重新取能力，不重放已消费 nonce。升级顺序为 control、node 与 agent 映射、web；整文件接口保持兼容。完整状态与错误语义见顶层“接口与一致性约定”。

## 运行日志与节点配置接口

迁移 16 新增运行日志、保留策略与节点配置表，17 新增日志策略实例确认；不修改文件归属、引用、任务或物理布局。升级先停服务并备份 PG 与节点根，运行 `control -config /path/control.json -migrate-only`，再升级各消费者。

私有 `log_keys` 是 `{web:"独立随机密钥",status:"另一个独立随机密钥"}`，各至少 32 字节，不能与其他安全密钥复用；只授权 `/internal/v1/logs`。实际密钥由部署工具生成，禁止使用文档中的占位文字。node/maintenance 沿用原内部签名。默认不提供任意文本入口；允许事件码为 started、stopping、request_failed、reconcile_failed、resource_report_failed、storage_corrupt、task_failed、connection_failed、connection_restored、command_completed、command_failed、configuration_applied、configuration_failed、logs_dropped、probe_failed、probe_recovered。CLI 仅允许 command_completed、command_failed、logs_dropped，不允许管理员令牌。发生时间允许过去 7 天至未来 5 分钟；接收时间由 PG 生成。

管理员 `GET /admin/v1/logs` 支持 before/level/module，倒序最多 100 条；`GET/POST /admin/v1/log-policy` 的提交字段为 version、info_limit（默认 1000，0..100000）、warning_days（默认 30，1..365）、upload_seconds（默认 10，2..60）、min_level（info/warn/error）。POST 完整校验、CAS 和清理原子提交；冲突 409、非法 400。缩短保留范围立即删除超限运行日志，不能恢复；不删除安全审计及任务状态。GET 附带 control/maintenance/web/status 活跃一天内实例的已确认版本、最近上报和 applied/pending/stale；150 秒未上报为 stale。节点独立配置日志级别/间隔；CLI 只在命令存续期间拉取默认策略。并非提交成功就已全部应用。

每批最多 100 条、64 KiB；按认证来源每分钟 6000 条服务事件、120 条 CLI 事件及每小时 128 个活跃 stream 限流。PG stream 高水位保留 8 天，重复 batch 幂等；日志使用独立短事务锁 904218、SQL 1.5 秒/锁 0.5 秒超时，与文件任务锁分开。故障不失败原业务；静默时每分钟清理。缓冲内记录在进程崩溃时可能丢失，不替代安全审计。

`GET /admin/v1/node-configurations` 返回配置列表和逐实例状态；`POST /admin/v1/node-configurations/{id}` 必须提交 `{version,settings}` 全量六字段。读写预算 0..10^12 字节/秒、保留空间 0..2^53-1 字节、resource_interval_seconds 1..5、log_upload_seconds 2..60、log_min_level info/warn/error。PG CAS 与调度列原子更新，新调度立即使用；节点每 3 秒签名拉取资源/日志设置并下一轮确认，20 秒无报告为 stale。现有预留、能力、在途任务和连接不中断，恢复旧值以最新版本重新提交。生产者不能替其他节点确认；根/后端格式/容量/密钥/连接不能在线修改。完整协议见顶层“接口与一致性约定”。
