import { Button, Progress, Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../ui';
import { FileIcon } from '../files/FileIcon';
import { bytes, date } from '../../lib/format';
import type { Transfer } from './state';
import { duration } from './metrics';

type Props = { items: Transfer[]; completed: boolean; retry(id: string): void; cancel(id: string): void };
export function TransferTable({ items, completed, retry, cancel }: Props) {
  return <Table className="transfer-table"><TableHeader><TableRow><TableHead>文件</TableHead><TableHead>大小</TableHead><TableHead>{completed ? '完成时间' : '操作'}</TableHead></TableRow></TableHeader><TableBody>
    {items.map(t => <TableRow className="transfer" key={t.id}><TableCell><div className="transfer-file"><FileIcon name={t.name} /><div className="transfer-details"><strong title={t.name}>{t.name}</strong>
      {!completed && <><small className={t.state === 'failed' ? 'error' : ''}>{t.label}</small><Progress max={100} value={t.progress} size="sm" aria-label={`${t.name} 传输进度`} /><small className="transfer-speed">速度：{t.state==='running'&&t.speed!==undefined?`${bytes(Math.round(t.speed))}/s`:'—'} · 预计剩余：{t.state==='running'?duration(t.remaining):'—'}</small></>}
    </div></div></TableCell><TableCell>{t.size === undefined ? '—' : bytes(t.size)}</TableCell><TableCell>{completed ? <time dateTime={t.completedAt}>{date(t.completedAt || null)}</time> : ['failed', 'cancelled'].includes(t.state) ? <Button variant="ghost" onClick={() => retry(t.id)}>重试</Button> : <Button variant="ghost" onClick={() => cancel(t.id)}>取消</Button>}</TableCell></TableRow>)}
  </TableBody></Table>;
}
