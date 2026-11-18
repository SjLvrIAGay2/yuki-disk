import {Badge,Table,TableBody,TableCell,TableHead,TableHeader,TableRow} from '../../ui';
import {ConfigurationHelp} from './ConfigurationHelp';
import {configurationStatus,type ConfigurationReport} from './configurationStatus';
export function ConfigurationServiceStatus({services,target}:{services:{name:string;reports:ConfigurationReport[]}[];target:number}){
 return <Table><TableHeader><TableRow>{['服务','配置应用状态','最近报告'].map(v=><TableHead key={v}>{v}</TableHead>)}</TableRow></TableHeader><TableBody>{services.map(service=>{const status=configurationStatus(service.reports,target);return <TableRow key={service.name}><TableCell>{service.name}</TableCell><TableCell><Badge variant={status.variant}>{status.label}</Badge><ConfigurationHelp label={`${service.name}配置应用状态`} content={status.detail}/></TableCell><TableCell>{status.observed?new Date(status.observed).toLocaleString():'尚无报告'}</TableCell></TableRow>})}</TableBody></Table>;
}
