import {useEffect,useRef,useState} from 'react';
import {Alert,Button,Select} from '../../ui';
import {api} from '../../api/client';
import {useAction,type AdminProps} from './common';
import {SystemConfigurationPanel} from './SystemConfigurationPanel';
import {LogPolicyPanel} from './LogPolicyPanel';
import {NodeConfigurationPanel,type NodeConfigurations} from './NodeConfigurationPanel';
export function ConfigurationPanel(props:AdminProps){
 const [target,setTarget]=useState('system'),[nodes,setNodes]=useState<NodeConfigurations>({items:[],instances:[]});const{busy,error,run}=useAction();
 const pending=useRef<Promise<void>|null>(null);
 function load():Promise<void>{
  if(pending.current)return pending.current;
  pending.current=(async()=>{try{setNodes(await api<NodeConfigurations>(props.config,props.token,'/admin/v1/node-configurations'))}catch(e){setNodes(old=>({...old,instances:old.instances.map(v=>v.state==='superseded'?v:{...v,state:'stale'})}));throw e}finally{pending.current=null}})();
  return pending.current;
 }
 useEffect(()=>{void run(load);const timer=setInterval(()=>void run(load),10000);return()=>clearInterval(timer)},[props.config,props.token]);
 return <section><h2>运行配置</h2>{error&&<Alert variant="danger" title={error}/>}<Select label="配置对象" value={target} onValueChange={v=>v&&setTarget(v)} options={[{value:'system',label:'系统默认值'},{value:'logs',label:'日志保留与上报'},...nodes.items.map(n=>({value:'node:'+n.id,label:'存储节点 '+n.id}))]}/>
 {target==='system'&&<SystemConfigurationPanel {...props}/>} {target==='logs'&&<LogPolicyPanel {...props}/>}
 {target.startsWith('node:')&&nodes.items.filter(n=>target==='node:'+n.id).map(n=><NodeConfigurationPanel key={n.id} {...props} node={n} instances={nodes.instances.filter(v=>v.node_id===n.id)} refresh={load}/>)}
 <Button disabled={busy} onClick={()=>void run(load)}>刷新节点配置状态</Button></section>;
}
