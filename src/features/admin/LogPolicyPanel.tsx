import {useConfigurationReports} from './useConfigurationReports';
import {ConfigurationServiceStatus} from './ConfigurationServiceStatus';
import {serviceNames} from './configurationStatus';
import {ConfigurationHelp} from './ConfigurationHelp';
import {useEffect,useState} from 'react';
import {Alert,Button,Input,Select} from '../../ui';
import {api} from '../../api/client';
import {useAction,type AdminProps} from './common';
type Policy={version:number;info_limit:number;warning_days:number;upload_seconds:number;min_level:string;instances:{source:string;id:string;applied_version:number;state:string;observed_at:string}[]};
export function LogPolicyPanel({config,token}:AdminProps){
 const [value,setValue]=useState<Policy>();const observed=useConfigurationReports<Policy['instances'][number]>({config,token},'/admin/v1/log-policy');const {busy,error,run}=useAction();
 async function load(){setValue(await api<Policy>(config,token,'/admin/v1/log-policy'))}useEffect(()=>{void run(load)},[config,token]);
 return <section><h3>日志保留与上报</h3>{error&&<Alert variant="danger" title={error}/>}{value&&<><form onSubmit={e=>{e.preventDefault();void run(async()=>{const {instances,...policy}=value;await api(config,token,'/admin/v1/log-policy',policy);await load()})}}>
 <div className="configuration-field"><Input label="普通日志保留条数" type="number" required min={0} max={100000} value={value.info_limit} onChange={e=>setValue({...value,info_limit:Number(e.target.value)})}/><ConfigurationHelp label="普通日志保留条数" content="保留最新的普通日志记录，默认 1000 条，可设为 0 至 100000。重复事件可能合并计数；设为 0 不保留普通日志。减少条数会立即清理超出范围的记录。"/></div>
 <div className="configuration-field"><Input label="警告和错误保留天数" type="number" required min={1} max={365} value={value.warning_days} onChange={e=>setValue({...value,warning_days:Number(e.target.value)})}/><ConfigurationHelp label="警告和错误保留天数" content="警告和错误按入库时间保留，默认 30 天，可设为 1 至 365 天，不计入普通日志条数。缩短期限会立即清理超期记录。"/></div>
 <div className="configuration-field"><Input label="默认上报间隔（秒）" type="number" required min={2} max={60} value={value.upload_seconds} onChange={e=>setValue({...value,upload_seconds:Number(e.target.value)})}/><ConfigurationHelp label="默认上报间隔（秒）" content="可设为 2 至 60 秒，在服务下一次成功上报后生效。存储节点使用各自的上报间隔；命令行在命令结束时补充上报。连接中断时缓冲有限数量的记录并退避重试。"/></div>
 <div className="configuration-field"><Select label="默认记录级别" value={value.min_level} onValueChange={v=>v&&setValue({...value,min_level:v})} options={[{value:'info',label:'普通、警告和错误'},{value:'warn',label:'警告和错误'},{value:'error',label:'仅错误'}]}/><ConfigurationHelp label="默认记录级别" content="只影响新产生的日志，在服务下一次成功上报后生效。普通包含所有级别，警告包含警告和错误；存储节点使用各自的日志级别。"/></div>
 <Button disabled={busy} type="submit">保存日志策略</Button></form>
 <ConfigurationServiceStatus target={Math.max(value.version,observed?.version??0)} services={['control','maintenance','web','status'].map(source=>({name:serviceNames[source],reports:(observed?.instances??value.instances).filter(v=>v.source===source)}))}/></>}<Button disabled={busy} onClick={()=>void run(load)}>重新读取日志策略</Button></section>;
}
