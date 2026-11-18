export type ConfigurationReport={id:string;state:string;applied_version:number;observed_at:string};
export const serviceNames:Record<string,string>={control:'控制服务',node:'存储服务',maintenance:'维护服务',web:'网盘网页服务',status:'状态页服务',cli:'命令行客户端'};
export function configurationStatus(reports:ConfigurationReport[],target:number){
 const current=reports.filter(v=>v.state!=='superseded');
 const applied=current.filter(v=>v.state==='applied'&&v.applied_version===target).length;
 const failed=current.filter(v=>v.state==='failed').length;
 const stale=current.filter(v=>v.state==='stale').length;
 const pending=current.length-applied-failed-stale;
 const label=!current.length?'尚未收到报告':failed?'应用失败':stale?(applied?'部分已应用，部分报告过期':'报告已过期'):pending?(applied?'部分已应用':'等待应用'):'已应用';
 const detail=!current.length?'等待服务首次报告。':`${applied} 项已应用，${pending} 项等待应用，${failed} 项失败，${stale} 项报告过期。${reports.length>current.length?'旧进程报告已归入历史。':''}`;
 const observed=reports.map(v=>v.observed_at).sort().at(-1);
 return {label,detail,observed,variant:failed?'danger' as const:applied===current.length&&current.length>0?'success' as const:'warning' as const};
}
