import {ConfigurationServiceStatus} from './ConfigurationServiceStatus';
import {ConfigurationHelp} from './ConfigurationHelp';
import {useEffect,useState} from 'react';
import {Alert,Button,Input,Select} from '../../ui';
import {api} from '../../api/client';
import {useAction,type AdminProps} from './common';
type Values={read_bandwidth_bps:number;write_bandwidth_bps:number;reserve_bytes:number;resource_interval_seconds:number;log_upload_seconds:number;log_min_level:string};
type NodeConfig={id:string;version:number;settings:Values;tier:string;storage_mode:string|null;capacity_bytes:number|null};
type Instance={node_id:string;id:string;target_version:number;applied_version:number;state:string;observed_at:string};
export type NodeConfigurations={items:NodeConfig[];instances:Instance[]};
const fields:{key:Exclude<keyof Values,'log_min_level'>;label:string;min:number;max:number;help:string}[]=[
 {key:'read_bandwidth_bps',label:'读取调度预算（字节 / 秒，0 为不设预算）',min:0,max:1e12,help:'影响新下载和维护任务的节点选择权重，不是网络限速；已有传输与预留继续有效。'},
 {key:'write_bandwidth_bps',label:'写入调度预算（字节 / 秒，0 为不设预算）',min:0,max:1e12,help:'影响新上传和迁移任务的节点选择权重，不中断已开始的任务。'},
 {key:'reserve_bytes',label:'保留空闲空间（字节）',min:0,max:Number.MAX_SAFE_INTEGER,help:'新任务分配空间时扣除该额度；已有空间预留不释放。超过当前空闲空间时将暂停向此节点分配新副本。'},
 {key:'resource_interval_seconds',label:'资源采样上报间隔（秒）',min:1,max:5,help:'应用后从下一轮采样开始使用。采样失败时继续遵守资源数据过期限制。'},
 {key:'log_upload_seconds',label:'日志上报间隔（秒）',min:2,max:60,help:'只影响此节点，当前上报完成后按新间隔执行；失败时有退避重试。'},
 ];
export function NodeConfigurationPanel({config,token,node,instances,refresh}:AdminProps&{node:NodeConfig;instances:Instance[];refresh():Promise<void>}){
 const[draft,setDraft]=useState(node),[dirty,setDirty]=useState(false);const{busy,error,run}=useAction();
 useEffect(()=>{if(!dirty)setDraft(node)},[node,dirty]);

 return <section><h3>存储节点 {node.id}</h3><p>{node.tier==='cold'?'冷存储':'热存储'} · {node.storage_mode==='container'?'预分配容器':node.storage_mode==='filesystem'?'文件系统':'存储信息暂不可用'}</p>{error&&<Alert variant="danger" title={error}/>} {dirty&&draft.version!==node.version&&<Alert variant="warning" title="配置已由其他操作更新，请重新读取后修改。"/>}
 <form onSubmit={e=>{e.preventDefault();void run(async()=>{await api(config,token,`/admin/v1/node-configurations/${encodeURIComponent(node.id)}`,{version:draft.version,settings:draft.settings});setDirty(false);await refresh()})}}>{fields.map(f=><div className="configuration-field" key={f.key}><Input required label={f.label} type="number" min={f.min} max={f.max} step={1} value={draft.settings[f.key]} onChange={e=>{setDirty(true);setDraft({...draft,settings:{...draft.settings,[f.key]:Number(e.target.value)}})}}/><ConfigurationHelp label={f.label} content={f.help}/></div>)}
 <div className="configuration-field"><Select label="节点日志级别" value={draft.settings.log_min_level} onValueChange={v=>{if(v){setDirty(true);setDraft({...draft,settings:{...draft.settings,log_min_level:v}})}}} options={[{value:'info',label:'普通、警告和错误'},{value:'warn',label:'警告和错误'},{value:'error',label:'仅错误'}]}/><ConfigurationHelp label="节点日志级别" content="只影响此节点新产生的日志。普通包含所有级别，警告包含警告和错误；已有日志仍按保留策略清理。"/></div><Button disabled={busy||draft.version!==node.version} type="submit">保存此节点配置</Button><Button type="button" disabled={busy} onClick={()=>{setDirty(false);setDraft(node)}}>重新读取节点配置</Button></form>

 <ConfigurationServiceStatus target={node.version} services={[{name:'存储节点 '+node.id,reports:instances}]}/>
 </section>;
}
