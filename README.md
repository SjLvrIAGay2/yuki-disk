# Yuki Disk CLI

Yuki Disk 命令行客户端仅供普通用户上传下载、目录管理、版本与回收站操作。文件内容直接与白名单存储节点传输，控制面负责登录、角色和归属校验。管理员登录和复用管理员令牌访问网盘接口由服务端拒绝。

上传遇到 `503 insufficient_capacity` 时，控制面可能已优先调度符合条件的永久删除块回收，但响应不表示空间已释放。客户端沿用有界重试与同一上传幂等身份，等待物理删除及新鲜空间报告；重试预算耗尽后明确失败，可稍后重新尝试。此机制不会释放回收站恢复期内或其他文件仍引用的数据。

上传副本数遵循控制面计划返回的目标列表，支持控制面显式选择单副本，无需 CLI 配置副本数。同一文件的不同块可落在不同节点，下载逐块校验并重组。单副本节点丢失会使其独有数据无法恢复。

元数据请求维持短超时，块上传和下载使用独立 120 秒超时，仍遵循原取消信号、节点白名单与禁止重定向规则。节点的 `403 capability_rejected` 表示本次短期能力不可再用，上传保留原幂等键和正文重新规划，最多八轮退避；其他 403 权限拒绝不自动重试。上传任务过期返回 410，需明确发起新任务，不能偷偷换键绕过过期状态。

## 安装与配置

原生运行于 Linux amd64、Windows amd64 或 macOS arm64，构建使用 Go 1.27.1，不要求 Linux 环境。在本仓库目录执行：

```bash
mkdir -p bin
go build -trimpath -o bin/yuki .
sudo install -m 0755 bin/yuki /usr/local/bin/yuki
```

Windows 使用 `go build -trimpath -o bin/yuki.exe .`，直接运行 `bin\yuki.exe`；macOS 可将 `bin/yuki` 放入用户自己的 PATH，无需以管理员权限运行。私有配置和下载目录由当前操作系统用户拥有。Windows 下载使用原生排他锁及目录刷盘，Unix 使用 `flock`；成功发布前完成文件与目录持久化，目标已存在或其他进程占锁时拒绝覆盖。

从部署管理员取得私有 `cli.json`，或参考 [config.example.json](config.example.json) 填写。生产配置字段如下，所有值均应来自实际部署：

| 字段 | 说明 |
|---|---|
| `control` | 控制面 HTTPS origin |
| `username` | 普通用户账号；旧配置省略时仅映射 control 显式 legacy_username |
| `password` | 该账号密码，12..1024 字节，私有配置保存 |
| `nodes` | 全部允许访问的节点 HTTPS origin 数组 |
| `development` | 生产为 `false`；回环 HTTP 仅可用于 `true` |

URL 必须是精确 origin，不含路径、尾斜线、查询、片段和凭据。配置为严格 JSON，无环境变量覆盖。使用组织 CA 时安装可信根，或通过 Go 的 `SSL_CERT_FILE` 指定信任根文件。

```bash
chmod 0600 /path/to/cli.json
yuki -config /path/to/cli.json ls
```

所有全局参数必须放在命令之前。默认配置路径为当前目录的 config.json。访问令牌只保存在进程内存，有效期五分钟；请求到期后单飞重新登录并只重试一次，保留原请求和幂等键。禁用、角色修改或重置旧凭据后不能继续登录；配置包含密码，不放网页静态目录、URL、日志或仓库。

## 常用操作

```bash
yuki -config /path/to/cli.json mkdir photos
yuki -config /path/to/cli.json upload ./photo.jpg photos/photo.jpg
yuki -config /path/to/cli.json ls photos
yuki -config /path/to/cli.json versions photos/photo.jpg
# VERSION_ID 使用上传或版本列表返回的 id。
yuki -config /path/to/cli.json download VERSION_ID ./saved-photo.jpg

yuki -config /path/to/cli.json mkdir archive
yuki -config /path/to/cli.json cp photos/photo.jpg archive
yuki -config /path/to/cli.json rename archive/photo.jpg cover.jpg
yuki -config /path/to/cli.json trash archive/cover.jpg
yuki -config /path/to/cli.json restore archive/cover.jpg
```

成功结果输出为 JSON，失败以非零退出码结束。远程路径使用相对路径和 `/` 分隔，不以 `/` 开头。

## 命令一览

| 命令 | 用途 |
|---|---|
| `upload LOCAL REMOTE_PATH` | 上传本地文件，返回 `version_id` |
| `download VERSION_ID TARGET` | 下载指定版本，目标必须不存在 |
| `ls [DIR]` | 列出目录，省略为根目录 |
| `mkdir PATH` | 创建目录 |
| `mv SOURCE TARGET_DIR` | 移动到已存在目录，保留历史版本 |
| `cp SOURCE TARGET_DIR` | 复制最新活动版本 |
| `rename SOURCE NAME` | 在同目录改名，NAME 不含路径分隔符 |
| `versions PATH` | 列出该路径活动和单独删除的版本 |
| `files` | 文件版本列表 |
| `trash PATH...` / `restore PATH...` | 同批删除或恢复路径，最多 100 项 |
| `recycle` | 按路径回收的文件版本 |
| `version-trash ID...` | 单独删除指定版本 |
| `version-recycle` | 列出单独回收的版本 |
| `version-restore ID TRASH_ID` | 恢复当前删除身份对应的版本 |
| `version-purge ID TRASH_ID` | 永久删除该回收版本，提前结束恢复期 |
| `mounts` | 查看只读服务器目录映射 |

移动、复制目标为目录，空字符串 `""` 表示根目录；同名冲突拒绝覆盖。目录复制包含活动文件的最新版本。恢复时原目录不可用会寻找最近可用上级，同名自动编号。`TRASH_ID` 取自当前版本列表，旧删除身份会被拒绝。

列表命令返回接口的第一页，不自动遍历所有分页。需要完整遍历时使用对应控制面 HTTP API 的 cursor/after 参数，详见[控制面 README](https://github.com/SjLvrIAGay2/yuki-disk/blob/control/README.md#http-接口)。按路径永久删除与清空回收站可使用网页。

## 重试与保存行为

上传及文件批次操作自动生成一次幂等键，内部重试复用。跨进程重试时显式提供相同键和相同请求：

```bash
yuki -config /path/to/cli.json -idempotency-key upload-photos-20260923 upload ./photo.jpg photos/photo.jpg
```

键长为 8..128 字节。不要把同一个键用于不同内容或操作；相同键改正文会冲突。已过期上传需重新开始并使用新键。

托管文件每块 4 MiB，单文件最多 10000 块。只有属于当前账号的既有内容或同代际有效上传证明才允许跳过字节；跨用户相同哈希仍需向节点 PUT 完整数据。节点签名证明与完整副本确认后才提交版本。下载逐块校验长度和 SHA-256，块失败时重新获取一次性能力，并尝试可用副本。

旧 visibility 命令明确提示旧链接失效；用户分享在 web 创建。block/events/tasks 系统诊断改为管理员 web，不为 CLI 增加管理会话。各接口仍独立执行服务端权限检查，不能改客户端绕过。

配置分类：control、nodes、username、password、development 与系统 TLS 信任均在每次 CLI 启动读取，运行中必须重启命令，不提供热修改监听；-idempotency-key 为新命令参数，不适用服务热配置。严格 JSON 为唯一应用配置，无 PG 运行配置或环境变量覆盖。已有任务保持原身份与请求；密码重置后更新私有配置，再以原幂等键重试仍有效的上传。

下载写入目标同目录的临时文件，同步后原子发布且不覆盖已有目标；同目标由内核文件锁互斥，拒绝符号链接父目录。持久隐藏锁文件存在不表示仍有任务占用，无需手动删除。SIGTERM 取消传输并清理本次临时文件；强制终止可能留下 `.part`，确认没有任务后可清理。

只读服务器文件使用目录列表返回的外部文件标识下载。仅管理员可在 web 创建挂载并授予指定普通用户，CLI 的 mounts 仅列出本用户已获授权目录；CLI 不提供主动挂载。标识绑定本用户与授权，不能交给其他用户复用。传输检查文件修订和大小，服务器文件发生变化时刷新目录后重新下载；schema 15 升级后旧未绑定用户的标识需刷新取得。撤销授权或停用账号后尚未开始的请求拒绝，已开始的整文件请求可能完成。该传输使用直接 HTTPS 整文件接口，CLI 不使用浏览器 P2P。节点地址不可达、证书不可信或不在白名单时会明确失败。

## 运行日志

普通用户命令成功登录后建立本进程缓冲，定时及命令结束时通过 `POST /v1/client-logs` 报告 command_completed/command_failed；不上传参数、路径、文件名、块操作、凭据或原始错误。管理员 CLI 登录及管理员会话复用仍由服务端拒绝。日志是经用户鉴权的客户端自报，不是受信任服务审计；管理员页明确标记来源。

缓冲最多 1000 条，默认间隔 10 秒，从响应热更新 2..60 秒及最低级别；每批最多 100 条，随机 stream 和 sequence 在安全重试中保持不变，普通会话失效仍需原凭据重新登录。退出最多补报 3 秒，网络失败不更改命令原退出码或幂等身份；进程崩溃可能丢失未发送事件。连接、安全材料、账号等启动配置仍需新建 CLI 进程生效，不写日志或仓库。
