import {useConfigurationReports} from './useConfigurationReports';
import {ConfigurationServiceStatus} from './ConfigurationServiceStatus';
import {ConfigurationHelp} from './ConfigurationHelp';
import {useEffect,useState} from 'react';
import {Alert,Button,Input} from '../../ui';
import {api} from '../../api/client';
import {useAction,type AdminProps} from './common';
type Values={trash_seconds:number;upload_seconds:number;user_share_seconds:number;api_share_seconds:number;login_attempts:number;download_authorizations:number};
type Runtime={version:number;settings:Values;all_observed_applied:boolean;instances:{id:string;applied_version:number;target_version:number;state:string;observed_at:string}[]};
const fields:{key:keyof Values;label:string;scale:number;min:number;max:number;help:string}[]=[
 {key:'trash_seconds',label:'回收站保留时间（天）',scale:86400,min:1/1440,max:90,help:'新放入回收站的文件和版本在期限内可恢复；期限届满后进入回收流程。已有回收期限不变。'},
 {key:'upload_seconds',label:'上传任务有效时间（分钟）',scale:60,min:1,max:1440,help:'从创建上传任务开始计时，超时后不能提交。只影响新任务，不延长正在上传或重试的任务。'},
 {key:'user_share_seconds',label:'用户分享默认时长（小时）',scale:3600,min:1/60,max:720,help:'创建用户分享时默认填入的时长，用户可在 30 天内调整。已有分享不变。'},
 {key:'api_share_seconds',label:'API 分享默认时长（小时，0 为永久）',scale:3600,min:0,max:876000,help:'创建来源授权和应用凭据时的默认值。永久应用仍要求永久来源授权；已有授权不变。'},
 {key:'login_attempts',label:'登录及分享验证次数（次 / 来源 / 分钟）',scale:1,min:1,max:120,help:'限制同一来源每分钟的登录及分享验证次数。调整后保留当前计数。'},
 {key:'download_authorizations',label:'分享下载授权次数（次 / 分享或应用 / 分钟）',scale:1,min:1,max:6000,help:'限制为分享或应用签发下载授权的频率。不是下载带宽限制，不中断已开始的块传输。'},
];
export function SystemConfigurationPanel({config,token}:AdminProps){const[value,setValue]=useState<Runtime>();const observed=useConfigurationReports<Runtime['instances'][number]>({config,token},'/admin/v1/configuration');const{busy,error,run}=useAction();
 async function load(){setValue(await api<Runtime>(config,token,'/admin/v1/configuration'))}useEffect(()=>{void run(load)},[config]);
 return <section><h3>系统默认值</h3>{error&&<Alert variant="danger" title={error}/>}<p>保存后请检查各项生效状态。未生效的服务继续使用原配置。</p>{value&&<><form onSubmit={e=>{e.preventDefault();void run(async()=>{await api(config,token,'/admin/v1/configuration',{version:value.version,settings:value.settings});await load()})}}>{fields.map(f=><div className="configuration-field" key={f.key}><Input label={f.label} type="number" required step={f.scale===1?1:"any"} min={f.min} max={f.max} value={value.settings[f.key]/f.scale} onChange={e=>setValue({...value,settings:{...value.settings,[f.key]:Math.round(Number(e.target.value)*f.scale)}})}/><ConfigurationHelp label={f.label} content={f.help}/></div>)}<Button type="submit" disabled={busy}>保存配置</Button></form><ConfigurationServiceStatus target={Math.max(value.version,observed?.version??0)} services={[{name:'控制服务',reports:observed?.instances??value.instances}]}/></>}<Button disabled={busy} onClick={()=>void run(load)}>刷新配置状态</Button></section>;
}
