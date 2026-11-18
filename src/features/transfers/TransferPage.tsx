import { useState } from 'react';
import { Button, EmptyState, Tabs, TabsList, TabsTrigger, TabsContent } from '../../ui';
import type { Transfer } from './state';
import { TransferTable } from './TransferTable';
import './transfers.css';

type Props = { transfers: Transfer[]; retry(id: string): void; cancel(id: string): void; clear(ids?: string[]): void };
export function TransferPage({ transfers, retry, cancel, clear }: Props) {
  const [direction, setDirection] = useState<'upload' | 'download'>('upload');
  const [view, setView] = useState('pending');
  const scoped = transfers.filter(item => (item.direction || 'upload') === direction);
  const pending = scoped.filter(item => item.state !== 'done');
  const completed = scoped.filter(item => item.state === 'done');
  const items = view === 'done' ? completed : pending;
  const clearable = items.filter(item => ['done', 'failed', 'cancelled'].includes(item.state)).map(item => item.id);
  return <section className="transfers" aria-label="传输任务">
    <aside className="transfer-sidebar"><nav aria-label="传输类型">
      {(['upload', 'download'] as const).map(value => <Button key={value} variant="ghost" className={direction === value ? 'active' : ''} aria-current={direction === value ? 'page' : undefined} onClick={() => setDirection(value)}>{value === 'upload' ? '上传' : '下载'}</Button>)}
    </nav></aside>
    <div className="transfer-main"><Tabs value={view} onValueChange={setView} variant="line">
      <TabsList aria-label="传输状态"><TabsTrigger value="pending">未完成 ({pending.length})</TabsTrigger><TabsTrigger value="done">已完成 ({completed.length})</TabsTrigger></TabsList>
      <div className="transfer-toolbar"><Button variant="outline" disabled={!clearable.length} onClick={() => clear(clearable)}>清除记录</Button><span>{items.length} 项</span></div>
      {['pending', 'done'].map(value => <TabsContent key={value} value={value}>
        {items.length ? <TransferTable items={items} completed={value === 'done'} retry={retry} cancel={cancel} /> : <EmptyState title="暂无传输任务" description={value === 'done' ? '完成的任务会显示在这里。' : `暂无未完成的${direction === 'upload' ? '上传' : '下载'}任务。`} />}
      </TabsContent>)}
    </Tabs></div>
  </section>;
}
