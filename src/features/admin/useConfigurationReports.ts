import {useEffect,useState} from 'react';
import {api} from '../../api/client';
import type {AdminProps} from './common';
import type {ConfigurationReport} from './configurationStatus';
// Observe acknowledgements independently of form drafts. Failed reads must not
// leave an old successful badge indefinitely; never overlap background reads.
export function useConfigurationReports<T extends ConfigurationReport>(props:AdminProps,path:string){
 const [observation,setObservation]=useState<{version:number;instances:T[]}>();
 useEffect(()=>{
  let active=true,pending=false;
  async function poll(){if(pending)return;pending=true;try{const next=await api<{version:number;instances:T[]}>(props.config,props.token,path);if(active)setObservation(next)}catch{if(active)setObservation(old=>old&&({...old,instances:old.instances.map(v=>({...v,state:'stale'}))}))}finally{pending=false}}
  void poll();const timer=setInterval(()=>void poll(),5000);
  return()=>{active=false;clearInterval(timer)};
 },[props.config,props.token,path]);
 return observation;
}
