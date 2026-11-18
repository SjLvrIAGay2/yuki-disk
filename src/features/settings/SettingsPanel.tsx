import { Button, Checkbox, Select, Accordion, AccordionItem, AccordionTrigger, AccordionContent, Alert, Card } from '../../ui';
import { useState, useSyncExternalStore } from 'react';
import { connectionSnapshot, subscribeConnection } from '../p2p/status';
import { clearCache } from '../transfers/cache';
import { clearDownloadStaging } from '../transfers/save';
import { defaultSettings, type Settings } from './settings';
import { sortOptions } from '../files/sorting';
import './settings.css';
const strengths = ['最低', '较低', '标准', '较高', '高', '最高'].map((label, i) => ({ label, value: String(i + 1) }));
export function SettingsPanel({ value, change }: { value: Settings; change(value: Settings): void }) {
  const [message, setMessage] = useState('');
  const connection = useSyncExternalStore(subscribeConnection, connectionSnapshot);
  return <Card className="settings-panel"><h2>文件浏览</h2>
    <Select label="默认排序" value={value.sort} options={sortOptions} onValueChange={v => change({...value,sort:v as Settings['sort']})} />
    <Select label="排序方向" value={value.order} options={[{value:'asc',label:'升序'},{value:'desc',label:'降序'}]} onValueChange={v => change({...value,order:v as Settings['order']})} />
    <Select label="列表密度" value={value.density} options={[{value:'comfortable',label:'舒适'},{value:'compact',label:'紧凑'}]} onValueChange={v => change({...value,density:v as Settings['density']})} />
    <Checkbox label="显示文件路径" checked={value.showPath} onCheckedChange={v => change({...value,showPath:v===true})} />
    <p>文件夹始终排在文件前。偏好保存在当前浏览器，刷新后继续使用。</p>
    <h2>传输设置</h2>
    <Checkbox label="传输完成时显示提醒" checked={value.notifyComplete} onCheckedChange={v => change({...value,notifyComplete:v===true})} />
    <p>下载交给浏览器保存到默认位置；如浏览器开启每次询问，仍会显示浏览器的保存提示。</p>
    <Checkbox label="加快重复下载" checked={value.cache} onCheckedChange={checked => change({ ...value, cache: checked === true })} />
    <p>临时文件在退出登录或刷新页面后清除，不影响网盘中的文件。</p>
    <Button variant="outline" onClick={() => void clearCache().then(() => setMessage('临时文件已清除')).catch(() => setMessage('清除未完成，请重试。'))}>清除临时文件</Button>
    <Button variant="outline" onClick={() => void clearDownloadStaging().then(()=>setMessage('闲置下载暂存已清除，正在下载的文件不受影响。')).catch(()=>setMessage('清理未完成，请稍后重试。'))}>清理下载暂存</Button>
    {message && <Alert title={message} />}
    <Accordion type="single" collapsible><AccordionItem value="transfer"><AccordionTrigger>更多传输设置</AccordionTrigger><AccordionContent>
      <Select label="连接方式" value={value.connection} onValueChange={v => change({ ...value, connection: v === 'http' ? 'http' : 'auto' })} options={[{ label: '自动（优先直连）', value: 'auto' }, { label: '常规连接', value: 'http' }]} />
      <p>自动模式无需安装程序。服务支持时优先直连，网络受限时自动使用中继。设置对之后开始的传输生效。</p>
      <p>最近连接：{{ none: '尚未传输', direct: '设备直连', relay: '中继连接', http: '常规连接' }[connection]}</p>
      <Select label="传输强度" value={String(value.concurrency)} onValueChange={v => change({ ...value, concurrency: Number(v) })} options={strengths} />
      <Checkbox label="自动调整下载并发" checked={value.adaptiveDownloads} onCheckedChange={v=>change({...value,adaptiveDownloads:v===true})}/>
      <p>自动下载根据实际表现调整，较大文件以较高强度开始；关闭后保持所选强度。较低强度可减少设备占用，上传保持固定强度。设置对之后开始的传输生效。</p>
    </AccordionContent></AccordionItem></Accordion>
    <Button variant="outline" onClick={() => {change({...defaultSettings});setMessage('已恢复默认设置');}}>恢复默认设置</Button>
  </Card>;
}
