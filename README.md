# Yuki Disk Maintenance

控制面的维护规划采用事务内批量任务/副本视图。执行器继续只消费 control 已提交的任务、租约和代际许可；不能自行依据缓存位置执行复制或删除，也不因规划批量化跳过确认与重试规则。

Yuki Disk 维护进程从控制面领取任务，执行副本修复、冷热迁移与过期内容回收，并提交执行结果。它不连接 PostgreSQL、不自行计算引用，也不传输文件字节；复制由存储节点直接完成。

目标副本数由 control 的启动配置决定，默认 2，也可显式选择 1；maintenance 不维护另一份副本数配置。降低副本数会通过已有任务流程逐步回收多余副本，节点每次物理操作仍须通过执行租约、代际与删除许可检查。单副本迁移仍先完成目标复制，再回收来源，不能直接删除唯一有效副本。

## 构建与启动

原生运行于 Linux amd64、Windows amd64 或 macOS arm64，构建使用 Go 1.27.1，不需要 WSL 或虚拟机。在本仓库目录执行：

```bash
mkdir -p bin
go build -trimpath -o bin/maintenance .
bin/maintenance -config /etc/yukidisk/maintenance.json
```

Windows 构建为 `bin/maintenance.exe`，使用 `bin\maintenance.exe -config C:\ProgramData\yukiDisk\maintenance.json` 启动；macOS 使用本机绝对配置路径。服务账户须能读取私有配置和信任证书，健康监听仍保持回环地址。下文 systemd 示例仅用于 Linux；各平台均沿用相同的任务租约、重试和执行者栅栏。

控制面及节点应已启动。配置字段见 [config.example.json](config.example.json)；生产配套配置由[项目部署流程](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#配置数据库与集群)生成，勿直接使用示例中的开发密钥。

## 配置

| 字段 | 说明 |
|---|---|
| `control` | 控制面精确 origin |
| `key` | 独立维护密钥，与控制面和节点的 `maintenance_key` 一致 |
| `nodes` | 允许执行任务的节点 origin 白名单 |
| `development` | 生产为 `false`；回环 HTTP 仅可用于 `true` |
| `listen` | 回环健康监听地址，默认 `127.0.0.1:8090` |
| `monitor_key` | 独立采集密钥，至少 32 字节，不得复用维护 key；缺省时拒绝诊断采集 |

使用 `-config` 指定严格 JSON，未知字段拒绝，无环境变量覆盖。URL 不含路径、尾斜线、查询或凭据；生产控制面和节点必须为 HTTPS。使用组织 CA 时通过操作系统信任库或 Go 的 `SSL_CERT_FILE` 提供信任根，不关闭证书验证。出站请求禁止重定向，单次超时为 10 秒。

配置文件权限为 `0600` 或 `0640 root:yukidisk`。维护身份具有数据管理权限，不向网页、普通客户端或状态页分发。修改配置后重启进程。

## 系统服务部署

准备无登录的 `yukidisk` 服务账户，将二进制和私有配置安装到服务路径：

```bash
sudo install -d -o root -g yukidisk -m 0750 /opt/yukidisk /etc/yukidisk
sudo install -o root -g yukidisk -m 0750 bin/maintenance /opt/yukidisk/maintenance
sudo install -o root -g yukidisk -m 0640 /path/to/maintenance.json /etc/yukidisk/maintenance.json
sudo install -m 0644 ../deploy/yukidisk-maintenance.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now yukidisk-maintenance
sudo journalctl -u yukidisk-maintenance -n 50 --no-pager
```

服务模板见 [yukidisk-maintenance.service](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/deploy/yukidisk-maintenance.service)。可在不同机器部署多个维护进程；同机实例需要独立配置、不同 `listen` 端口和独立 systemd 服务名。

## 多实例与任务恢复

每个进程启动时生成自己的 owner。控制面管理全局 20 秒领导租约及 15 秒任务租约，同时核对 owner、fence 和有效期限。备用实例收到 409 后等待，不会同时获得有效执行权。

执行流程为控制面 `claim` → 节点 `task` → 节点向控制面请求 `permit` → 维护向控制面 `complete`。内部请求使用独立 HMAC 身份签名，节点物理操作还受持久代际和互斥约束。

任务失败由控制面统一进行有限重试与退避；进程崩溃或停止后，未完成任务在租约到期后可以接管。过期执行者不能提交结果。不要通过手工修改数据库任务行或删除节点文件来代替接管。

## 健康与状态

```bash
# 使用实际配置中的回环端口。
curl --fail http://127.0.0.1:8090/readyz
# /statusz 和 /metrics 需由采集器安全注入 X-Yuki-Monitor。
```

- `/healthz`：进程活性。
- `/readyz`：最近 30 秒内是否收到控制面有效联系结果（包括备用实例的 409）。
- `/metrics`：最近联系时间等 Prometheus 指标。
- `/statusz`：进程与系统资源、等待/执行/失联状态、此次启动已确认的完成和失败任务计数。

等待表示没有正在执行的任务，不代表某个固定的备用角色。计数仅在完成回报获确认后更新，重启归零。粗粒度运行日志缓冲并集中入库，警告/错误另输出脱敏 stderr。

健康端口仅绑定本机，不直接开放公网。远程状态页采集时，用受控 HTTPS 反向代理转发 `/readyz` 和 `/statusz`，并限制采集来源。SIGTERM 会取消在途请求；一致备份时应先停止全部维护实例，恢复时在控制面和节点就绪后再启动。完整流程见[项目 README](https://github.com/SjLvrIAGay2/yuki-disk/blob/main/README.md#日常运维与恢复)。

所有字段 control、key、nodes、development、listen、monitor_key 均为必须重启配置：它们绑定调度信任、固定目标、监听及采集身份。上述启动字段无热修改或环境变量覆盖；日志周期与最低级别从 PG 日志策略响应热更新。重启不更改 PG 任务权威、租约栅栏和原幂等身份；过期执行者不能通过新连接继续提交旧任务。普通用户无维护任务查询或执行权限；web 管理员页显示集中运行日志；原任务事件读取接口继续受管理员鉴权。匿名 status 只读取聚合缓存，不能触发本进程任务。

账号删除的 30 天保留、恢复及到期元数据清理由 control 在 PG 权威事务中处理；通常引用归零的块经过上传保留期和 GC 延迟。容量压力下 control 可将无版本引用、无有效上传引用、无未确认预留及无复制任务的零引用块提前调度为 `delete / capacity_gc`，领取时优先于普通队列，不抢占执行中任务。本进程透传该原因，继续通过原领取、许可、执行及完成协议操作节点；不新增直接删除用户目录的路径，也不删除节点 exports 外部源文件。其他用户仍引用的块和恢复期内的数据保留。

运行日志复用 maintenance 签名调用 `POST /internal/v1/logs`。每进程缓冲最多 1000 条、每批最多 100 条，默认 10 秒上报；相邻失败合并计数，不记录成功单块维护动作。任务失败只上传 task_failed，不上传哈希、任务内容或异常。批次 stream/sequence 在重试中不变，内部 nonce 重新生成；收到 ack 后出队。失败指数退避至 60 秒，退出限时补报，日志网络/数据库故障不改变任务完成协议。SIGKILL 可能丢失未发送缓冲。普通日志和警告/错误分别采用管理员配置的条数/天数保留策略，默认 1000/30。
