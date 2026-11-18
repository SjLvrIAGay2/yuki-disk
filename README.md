# Yuki Disk Node

Yuki Disk 存储节点负责内容块的持久保存、完整性校验、上传下载、节点间复制和物理回收，也可提供服务器目录的只读访问。节点不连接数据库，位置分配和任务许可由控制面决定。

控制面容量紧张时可能提前下发 `delete`、`reason=capacity_gc`。该原因使用与普通 GC 相同的节点签名、在线许可、代际墓碑、物理互斥和后端持久删除；不删除挂载 exports 源文件、不变更存储格式。预分配容器只释放已提交的内部页，容器文件不缩小。删除成功后的资源采样重新报告可分配空间，不能依据排队任务提前增加空闲容量。

## 构建与启动

使用 Linux amd64、Windows amd64 或 macOS arm64 原生进程，构建使用 Go 1.27.1。在本仓库目录执行：

```bash
mkdir -p bin
go build -trimpath -o bin/node .
bin/node -config /etc/yukidisk/node.json
```

使用本地 ext4（Linux）、APFS（macOS）或 NTFS（Windows），不使用网络共享目录作为块存储。数据目录必须是绝对真实路径，不允许符号链接；Unix 上归服务账户所有且权限为 `0700`，Windows 上以 ACL 限制到服务账户及系统管理账户。同一数据根只允许一个进程，锁冲突会直接拒绝启动。macOS 的 `/tmp`、`/var` 可能经过系统符号链接，配置应使用解析后的真实路径，不放宽路径校验。

Windows 构建与前台启动：`go build -trimpath -o bin/node.exe .`，然后执行 `bin\node.exe -config C:\ProgramData\yukiDisk\node.json`。macOS 可直接执行 `bin/node -config /Users/服务账户/Library/Application\ Support/yukiDisk/node.json`。JSON 中 Windows 路径的反斜线需写成 `\\`；也可使用 `C:/ProgramData/yukiDisk/data`。配置、证书及数据均需保存在受限目录，原生进程不需要 WSL、虚拟机或 Linux 容器。

完整配置示例：[目录模式](config.example.json)、[容器模式](config.container.example.json)。示例使用回环地址与开发标识，生产配套配置可通过[项目部署步骤](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#配置数据库与集群)生成。

## 配置

配置来自 `-config` 指定的严格 JSON，无环境变量覆盖，修改后重启。

| 字段 | 说明 |
|---|---|
| `id` | 控制面静态登记的节点 ID |
| `listen` | 本节点监听地址和端口 |
| `data` | 本地数据根绝对路径 |
| `control` | 控制面精确 origin |
| `key` | 本节点独立身份密钥，与控制面对应节点项一致 |
| `capability_key` | 数据能力密钥，与控制面一致 |
| `maintenance_key` | 维护身份密钥，与控制面、维护进程一致 |
| `development` | 生产为 `false`；回环 HTTP 仅可用于 `true` |
| `tls_cert` / `tls_key` | 生产必填的证书与私钥路径 |
| `origins` | 网页和跨站播放器 origin 白名单 |
| `sources` | 允许复制来源的节点 origin 白名单 |
| `storage_mode` | `filesystem`（默认）或 `container` |
| `capacity_bytes` | 容量配置，整数字节 |
| `container_file` | 容器文件名，仅容器模式使用，默认 `blocks.pack` |
| `exports` | 可选只读目录白名单，默认空；管理员仅能在此范围内建立网盘目录授权，修改根需重启 |

三类密钥至少 32 字节且互不相同。origin 不含路径、尾斜线、查询、片段和凭据。生产 URL 使用 HTTPS；服务账户须能读取证书私钥。不要将运行数据根授权给其他用户写入。

节点冷热角色在控制面 `nodes` 中设置；读写调度预算和预留余量从该配置首次导入 PG，之后由管理员按节点修改。本地路径和容量只在节点启动配置中设置。

块 PUT/GET 只有在签名能力、过期和持久化重放校验通过后才设置 120 秒读写截止时间，适应慢速链路的 4 MiB 块。能力通常 60 秒内准入，已经开始的单块可完成；不延长维护任务租约，不改变代际、块互斥或完整性校验。反向代理必须保留这一窗口并禁止写入自动重放；未认证请求、请求头和元数据仍受短超时约束。

macOS 访问网络宗卷需要本机隐私授权。首次启动以及更新未采用稳定签名身份的二进制后，系统可能重新请求“网络宗卷”权限；仅允许对应节点程序的必要访问，不关闭隐私保护或默认授予完整磁盘访问。SSH 隧道的 `UserKnownHostsFile` 如包含空格，传给 SSH 的选项值本身也必须用双引号包围；仅在 shell 层引用整个参数不足以避免 SSH 将路径解释为多个文件。

## 存储方式

### 目录模式

以下字段合并到完整配置：

```json
{
  "data": "/var/lib/yukidisk/node",
  "storage_mode": "filesystem",
  "capacity_bytes": 107374182400
}
```

每块一个文件。`capacity_bytes=0` 使用所在卷的可用空间，正数限制已存块有效载荷总量，最大 1 TiB。该限制不包含文件系统元数据、墓碑、防重放记录和临时文件，不是文件系统硬配额；实际磁盘可用空间仍构成上限。可调整配额，但不能低于已有有效载荷总量。

### 容器模式

```json
{
  "data": "/var/lib/yukidisk/node",
  "storage_mode": "container",
  "capacity_bytes": 107374182400,
  "container_file": "blocks.pack"
}
```

容器容量为 16 MiB 至 1 TiB，使用 Linux `fallocate`、macOS `F_PREALLOCATE/F_ALLOCATEALL` 或 Windows `FileAllocationInfo` 实际预分配，失败即拒绝启动，不降级为稀疏文件。重新打开时同时检查文件大小、硬链接数量及实际分配；Windows 还拒绝稀疏、压缩及 reparse 文件。块内容与位置索引保存在容器内，以 64 KiB 页分配；小块至少占一页，删除后复用页，容器文件不会收缩。文件头和索引占用部分容量，可用量同时受空闲页与索引槽限制。

macOS 对文件和目录提交使用完整刷盘；Windows 通过固定目录句柄重新申请写权限后刷盘，不依赖文件名重新定位目录，也不忽略访问拒绝。Windows/macOS 的容器文件名不能通过大小写与内部元数据目录混淆；Windows 同时拒绝保留设备名、额外数据流及尾随点/空格。

`container_file` 只能是文件名，不带目录。数据根还保留 `state/`、`nonces/`、`tmp/` 和格式标记。已有根不能直接切换后端，已有容器不能原地改变大小或名称；应以新目录、新节点加入集群后迁移数据。

两种方式共用鉴权、互斥、代际隔离和盘面对账。目录写入通过临时文件、同步和原子重命名发布；容器通过写时复制及双份校验描述符提交，删除描述符持久化后才复用页。无法安全解释的损坏状态会拒绝打开，不能将容器校验视为多副本的替代。

## 只读服务器目录

在节点完整配置中加入：

```json
{
  "exports": [{"id":"media","root":"/srv/media"}]
}
```

`root` 必须是所在操作系统上已存在的绝对目录，服务账户具有读取及目录遍历权限。根路径与读取的各级路径均禁止符号链接或 reparse point；FIFO、设备等特殊文件不列出、不读取。Windows 可使用 `E:/Media`，macOS 可使用真实挂载路径 `/Volumes/Media`。外部目录与本地块存储根分别配置，不得把网络挂载目录用作块存储根。再按[控制面说明](https://github.com/SjLvrIAGay2/yuki-disk/blob/control/README.md#只读目录与公开访问)添加目录授权。

节点不修改或导入外部文件。外部目录不计入块配额，不提供托管副本、历史版本或回收站，需由服务器自行备份。目录搜索只匹配当前层名称，单目录最多扫描 100000 项，超出应拆分目录。服务器更新文件建议使用临时文件加原子重命名；持续原地修改可能使下载中止，刷新列表后重新请求。

私有 `/v1/content` 使用一次性文件读取能力并核对归属授权，支持 HEAD、单/多 Range、ETag 和 If-Range，每节点最多 16 个并发整文件流。旧匿名 `/public/v1/content` 在多用户升级后明确失效，控制面拒绝旧公开授权。新用户/API 分享采用固定版本逐块下载。整文件接口需可达 HTTPS 节点入口。

## 安装为系统服务

在节点机器创建 `yukidisk` 服务账户，并安装二进制、配置和证书：

```bash
sudo install -d -o root -g yukidisk -m 0750 /opt/yukidisk /etc/yukidisk
sudo install -d -o yukidisk -g yukidisk -m 0700 /var/lib/yukidisk/node
sudo install -o root -g yukidisk -m 0750 bin/node /opt/yukidisk/node
# /path/to/node.json 是生成器为本节点生成的配置。
sudo install -o root -g yukidisk -m 0640 /path/to/node.json /etc/yukidisk/node.json
sudo install -m 0644 ../deploy/yukidisk-node.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now yukidisk-node
sudo journalctl -u yukidisk-node -n 50 --no-pager
```

服务模板见 [yukidisk-node.service](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/deploy/yukidisk-node.service)。自定义 `data` 时同步调整 `ReadWritePaths`；`ProtectHome=yes` 会阻止访问 home 中的目录。使用非特权业务端口或配置必要的绑定权限。应先启动控制面，再启动节点和维护。

## Windows 原生常驻运行

`deploy/windows` 提供可通过 SSH 执行的安装、任务注册与验证脚本，不要求 WSL。安装脚本接收受限路径中的私有配置、证书、私钥和预先核验的二进制 SHA256；拒绝覆盖已有安装根，使用 `-InstallRoot` 指定本地 NTFS 安装目录，`-CapacityBytes` 指定与私有配置一致的容器容量。首次启动由节点实际预分配；空间不足或分配失败不降级。运行前必须具备匹配的控制面登记、生产密钥、TLS 和传输入口。

任务使用 Windows 内置 LocalService（SID S-1-5-19），不额外创建账户密码、不要求用户登录。程序及配置仅允许服务身份读取，数据目录允许修改；SYSTEM 与本机管理员保留管理权限。任务在系统启动时运行，失败一分钟后重试，拒绝重叠实例；安装及注册本身不自动启动。`Install.ps1`、`Register-Task.ps1` 需要已有管理员会话，日常节点执行使用受限身份。

任务包装器在该进程范围使用 RemoteSigned，不修改机器或组织执行策略；组织要求签名时应签署脚本，不能降低组织策略。LocalService 与其他使用同一 OS 身份的本机服务共享身份边界，服务器应只安装可信服务，应用用户之间的隔离仍由控制面 PG 授权决定。

`Run-Tunnel.ps1` 可用 Windows OpenSSH 建立固定 SSH 反向隧道；独立私钥、固定主机公钥及 tunnel.json 存放在安装根的 private 目录，不进入源码或制品。私钥必须由操作者按部署说明生成，公网专用账户只允许指定回环端口的 remote forwarding，禁止 shell/TTY；主机身份不能以关闭 StrictHostKeyChecking 的方式绕过。隧道不终止节点 TLS，公网反向代理仍须验证节点证书。运行中的包装器在连接退出后等待 15 秒再连接，连接建立超时为 10 秒，保活为 15 秒、最多连续 3 次无响应；远端重启不依赖计划任务重新触发。配置错误在启动时拒绝，重连不会变更凭据、主机公钥或转发目标。停止隧道须停止整个计划任务，只结束 SSH 子进程会触发重连。

`Verify.ps1 -InstallRoot <安装目录> -PublicOrigin <节点HTTPS源>` 读取私有监控凭据但不打印，检查容器实际分配、大小、唯一硬链接及非稀疏/压缩/reparse 属性，并通过正常 TLS 验证读取就绪状态。这不替代真实上传下载、多副本及重启对账验收。

停止前暂停新传输并等待在途任务结束，再按精确名称停止任务。Windows 任务停止按进程中断处理，不承诺 Unix SIGTERM 式排空；重启须等待对账。长期停机需同时禁用启动任务。移除计划任务不删除数据；不得递归清理数据根来处理启动失败。

## macOS 原生常驻运行

将节点数据放在本地 APFS 卷。通过 launchd 以专用服务账户运行节点，ProgramArguments 指定节点程序及私有配置的绝对路径。只读外部目录单独登记在 `exports`。服务账户需要配置读取权限、数据根写入权限及所需外部目录的读取权限。

## 健康、对账与恢复

对账错误通过固定原因代码写入缓冲日志，包括超时、取消、控制服务拒绝、连接中断和网络失败，无法分类时保留一般对账失败。日志及 stderr 不输出原始错误正文。先升级 control 的日志代码白名单及 web 管理页，再更新节点；较旧节点继续发送原有代码。

就绪探测读取最近一次已完成对账的原子快照，不等待上传、磁盘提交或对账网络请求持有的存储锁。启动未完成对账、对账失败或最近成功结果超过 30 秒均返回未就绪；失败完成后立即发布，不延用之前的成功。此快照仅用于监控，不替代数据请求的鉴权、存储互斥、代际校验及持久提交条件。

`/healthz` 提供活性，`/readyz` 提供就绪状态；`/metrics` 与 `/statusz` 必须提供独立 monitor_key 对应的 X-Yuki-Monitor，未配置时拒绝，另限制监控网络。健康成功不证明所有副本完整。日志输出到 stderr，SIGTERM 最多排空 HTTP 10 秒。

## 上传归属与配置分类

上传能力可包含 upload_id。节点复用既有块互斥、持久代际与完整字节哈希验证，PUT 成功后签名报告 `/internal/v1/upload-proof {upload_id,hash,generation}`；控制面确认当前上传清单和副本后记录证明。上报失败返回可重试状态，客户端复用原上传身份重新获取能力。即使本地已存在相同块，也不能不读取实际上传字节就替新账号证明所有权。此过程不改变后端格式、预分配或容器描述符提交。

新版节点将增量副本与上传证明合并为一次签名 `POST /internal/v1/upload-receipt {upload_id,hash,size,generation}`，仅在完成原有物理持久化后发送，控制面将二者原子提交。完整副本条件、块互斥、全量盘点及代际栅栏不变。旧控制面返回 404 时回退到原 report、upload-proof 两步；超时或其他错误不回退，而是返回可重试结果，客户端继续复用原上传身份。能力不含 upload_id 的兼容调用仍使用原增量报告。应先更新控制面，再更新节点。

复制任务的网络接收不持物理操作锁，每节点最多接收一个、至多 4 MiB 的块。来源仍受固定白名单与能力约束；接收前验证任务许可，接收后验证完整大小/哈希，持锁再次验证 PG 租约、执行者栅栏及当前代际，并检查本地持久代际后才提交。重试复用原任务，未提交缓冲区在崩溃后自然丢弃，不改变后端格式或页面复用规则。GET 读取与完整验证仍持锁，网络发送使用独立不可变缓冲区；最多八个同时发送的块（32 MiB），超出返回可重试 503。已授权并读取的响应可在源块删除后完成，之后的新请求不会读取被删块。全量盘点、PUT 和删除仍保持原物理互斥。

| 配置 | 类别 | 原因 |
|---|---|---|
| id、data、storage_mode、capacity_bytes、container_file | 必须重启并遵守原格式/根校验 | 身份、根、配额和布局不热迁移；已有根不自动改格式，预分配失败不降级 |
| control、control_address、control_ca_file、key、capability_key、maintenance_key、monitor_key | 必须重启并协调信任配置 | 固定目的地、专用 CA、身份、签名和独立采集凭据，查询不回显 |
| listen、development、origins、sources、tls_cert、tls_key | 必须重启 | 监听、来源白名单、TLS 信任固定 |
| exports 全部字段 | 必须重启 | 文件系统根验证及权限边界 |
| crash_after_rename | 不适用生产热配置 | 仅开发故障注入，生产拒绝 |

启动字段来自严格 JSON，无环境变量覆盖。管理员“运行配置”选择节点后可修改六项 PG 运行设置：读取/写入调度预算（0..1000000000000 字节/秒，0 不设预算）、保留空间（0..9007199254740991 字节）、资源采样周期（1..5 秒）、日志上传周期（2..60 秒）、日志级别（info/warn/error）。预算是新任务调度权重，不是限速；保留空间只约束新预留。PG 持久值优先于启动预算，重启不回退；根、格式、身份、密钥、容量等仍需重启和原存储校验。

节点每 3 秒以自身签名调用 `/internal/v1/node-configuration`，发送 `{instance_id,applied_version,state}`，验证完整目标后原子应用；下一轮确认版本，失败保持原配置。管理员按每实例查看应用状态，20 秒未报告显示过期，旧实例保留一天。配置传播期间已有采样、连接及传输完成后使用新周期；过期资源仍不能用于新写分配。任务能力、代际、块互斥及事务空间预留不变。

日志进程缓冲最多 1000 条，周期调用签名 `/internal/v1/logs`，使用固定事件码和 stream/sequence 重试去重。相邻同类事件合并次数，不记录成功单块读写、删除；警告/错误同时输出脱敏 stderr。网络失败退避至最多 60 秒，SIGTERM 限时补报，SIGKILL 可能丢失未送达缓冲。日志上报失败不会改变节点物理操作结果；集中保留策略由控制面管理。

节点独立发送签名资源样本，通常每 3 秒一次；全量盘面清单按 1000 项分页，只有完整扫描结束才应用。重启后重新校验盘面，随后轮转校验内容；控制面离线时保留已有数据并重试报告，不凭失联自行删除块。

容量页面中的有效载荷与占用空间口径不同：容器按存储页计算占用，目录配额按有效载荷计算；同卷节点不能简单相加。预分配失败应检查实际空闲空间、文件系统支持和权限；格式不符应恢复正确配置，不能删除格式标记强行打开。

备份前停止维护与数据写入，停机保存整个 `data`，包括容器、墓碑、防重放记录及格式标记，并与同批 PostgreSQL 和密钥一起保存。恢复后等待对账，由剩余健康副本修复缺失；全部副本同时丢失无法恢复。完整操作见[备份与恢复](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#日常运维与恢复)。

## Administrative export protocol

挂载范围读取使用 `GET /v1/content/range`。请求需 Bearer 与精确单段 Range，禁止 If-Range，长度最多 4 MiB。节点签名回查控制面，核对账号和目录授权、实际大小及修订，持久消费单次 nonce；整文件能力不能用于范围入口，范围能力不能用于整文件入口。每次文件读取前后保持修订检查，复制缓冲为 256 KiB，减少网络宗卷的细粒度 stat 往返。范围的响应为 206、匹配 ETag/Content-Range；断流或文件变化不能被客户端当作完整下载。旧 `/v1/content` 保留。

需要避开控制面公网代理延迟时，可同时设置 `control_address:"192.0.2.10:443"` 和 `control_ca_file:"/absolute/private/ca.pem"`，并将 `control` 指向证书包含的 HTTPS 主机名及部署前缀。Windows 使用真实绝对路径，例如 `E:/private/ca.pem`。该 IP 是文档示例，必须替换为实际固定地址。私有 CA 和固定拨号仅用于控制面签名请求，拒绝其他目的地及重定向；不关闭主机名验证，不影响复制源连接。修改需重启，未配置时沿用标准信任及 DNS。部署先提供受验证的元数据入口，再更新节点；回滚同时恢复原 control URL 和这两个字段。

The control-signed POST `/internal/v1/export` also accepts an empty export and path with `list:true` to return only configured export IDs. It never returns host root paths. All existing signature, replay, directory traversal, symlink and file revision checks remain mandatory. User mount grants are PostgreSQL metadata controlled by administrators; file-read capabilities bind the ordinary owner and grant, and the node rechecks authorization with control before opening content. Revoked grants reject requests that have not begun; an already running whole-file response can finish. Removing grants or deleting an account never deletes export source files. Export roots still require node configuration and restart; the admin page cannot select arbitrary host paths.
