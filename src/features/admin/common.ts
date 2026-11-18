import { useState } from 'react';
import type { Config } from '../../api/protocol';
import { displayError } from '../../lib/errors';
export type AdminProps = { config: Config; token: string };
export type Account = {id:string;username:string;role:string;disabled:boolean;version:number;deleted_at:string|null;purge_after:string|null};
export function useAction() {
 const [busy,setBusy]=useState(false),[error,setError]=useState('');
 async function run(action:()=>Promise<unknown>){setBusy(true);setError('');try{await action()}catch(e){setError(displayError(e,'操作未完成，请刷新后重试。'))}finally{setBusy(false)}}
 return {busy,error,run};
}
