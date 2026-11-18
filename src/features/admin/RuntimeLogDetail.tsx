import {Dialog,Badge} from '../../ui';
import {serviceNames} from './configurationStatus';
export type LogRecord={id:number;module:string;source:string;level:string;code:string;occurred_at:string;received_at:string;count:number};
const hints:Record<string,string>={
 reconcile_timeout:'对账请求未在期限内完成。检查控制服务、节点连接和数据库负载；后续成功对账后节点可恢复就绪。',
 reconcile_cancelled:'对账请求被取消，可能发生在服务停止或请求期限结束时。结合相邻的启动、停止和恢复事件判断。',
 reconcile_control_auth_rejected:'控制服务拒绝了节点认证。检查双方配置的节点身份、凭据和系统时间。',
 reconcile_control_conflict:'对账状态与当前记录冲突。节点会重新对账；持续出现时检查是否存在重复节点身份。',
 reconcile_control_rate_limited:'控制服务限制了请求频率。节点会按既有机制重试。',
 reconcile_control_unavailable:'控制服务暂时无法完成对账。检查控制服务和数据库的运行状态。',
 reconcile_control_rejected:'控制服务拒绝了对账请求。检查节点与控制服务的配置是否一致。',
 reconcile_connection_closed:'对账连接在完成前被关闭。检查网络连接与服务重启记录。',
 reconcile_network_failure:'节点未能通过网络完成对账。检查节点到控制服务的连接。',
 configuration_failed:'节点拒绝了收到的运行配置并继续使用原配置。请检查该节点的配置应用状态。',
 configuration_applied:'节点已通过校验并应用运行配置。配置页展示最新的应用报告。',
 probe_failed:'一次服务健康采集失败或返回异常。此记录未保存具体探测目标；请结合系统运行状态查看。',
 probe_recovered:'健康采集恢复成功。此事件表示当次采集结果恢复，不保证后续持续可用。',
 logs_dropped:'缓冲区容量不足，部分日志未能保留。可检查连接与日志上报间隔。',
 storage_corrupt:'节点发现存储完整性异常。应检查节点存储及相关恢复记录。',
};
export function RuntimeLogDetail({record,title,onClose}:{record:LogRecord;title:string;onClose():void}){
 const level=record.level==='error'?'错误':record.level==='warn'?'警告':'普通';
 return <Dialog open onOpenChange={open=>{if(!open)onClose()}} title="运行日志详情"><h3>{title}</h3><Badge variant={record.level==='error'?'danger':record.level==='warn'?'warning':'neutral'}>{level}</Badge><dl>
 <dt>服务</dt><dd>{serviceNames[record.module]||record.module}</dd><dt>来源</dt><dd>{record.source}</dd>
 <dt>发生时间</dt><dd>{new Date(record.occurred_at).toLocaleString()}</dd><dt>接收时间</dt><dd>{new Date(record.received_at).toLocaleString()}</dd>
 <dt>合并次数</dt><dd>{record.count}</dd><dt>事件代码</dt><dd>{record.code}</dd><dt>记录编号</dt><dd>{record.id}</dd></dl>
 <p>{hints[record.code]||'服务上报了此运行事件，未记录更具体的原因。可结合相邻记录与当前运行状态排查。'}</p>
 {record.count>1&&<p>相邻重复事件已合并，发生时间为本组首次记录时间。</p>}
 {record.module==='cli'&&<p>此事件由客户端报告，未经服务端独立验证。</p>}
 </Dialog>;
}
