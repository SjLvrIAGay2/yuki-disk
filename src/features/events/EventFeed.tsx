import { useEffect, useState } from 'react';
import { Alert, EmptyState, Progress, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../ui';
import { APIError, request } from '../../api/client';
import type { Event, EventPage } from '../../api/protocol';

function label(event: Event) {
  if (event.kind === 'placement_scheduled') return '自动分流已安排数据迁移';
  if (event.kind === 'copy_completed') return ({ cold_migration: '冷存储迁移完成', hot_migration: '热点数据迁移完成', capacity_rebalance: '容量均衡迁移完成', bandwidth_rebalance: '带宽均衡迁移完成' } as Record<string, string>)[event.reason || ''] || '存储副本修复完成';
  if (event.kind === 'delete_completed' && event.reason === 'migration_trim') return '迁移后的旧副本已清理';
  return ({ delete_completed: '存储空间回收完成', block_available: '文件数据已就绪', orphan_discovered: '发现待整理数据' } as Record<string, string>)[event.kind] || '维护状态已更新';
}
export function EventFeed({ token, onExpired }: { token: string; onExpired(): void }) {
  const [items, setItems] = useState<Event[]>([]); const [error, setError] = useState(false); const [syncing, setSyncing] = useState(true);
  useEffect(() => {
    const controller = new AbortController(); let timer: ReturnType<typeof setTimeout>; let after = 0;
    async function load() {
      let delay = 10000;
      try {
        const page = await request<EventPage>(`/api/events?after=${after}`, { token, signal: controller.signal });
        if (controller.signal.aborted) return;
        after = page.nextAfter; setItems(old => [...page.items.reverse(), ...old].slice(0, 1000));
        setError(false); setSyncing(page.more); if (page.more) delay = 200;
      } catch (cause) {
        if (controller.signal.aborted) return;
        if (cause instanceof APIError && cause.status === 401) { onExpired(); return; }
        setError(true); setSyncing(false);
      }
      if (!controller.signal.aborted) timer = setTimeout(load, delay);
    }
    void load(); return () => { controller.abort(); clearTimeout(timer); };
  }, [token, onExpired]);
  return <>{error && <Alert variant="danger" title="维护记录暂时无法更新，将自动重试。" />}{syncing && <Progress value={null} label="正在同步维护记录…" />}
    {items.length > 0 ? <Table><TableHeader><TableRow><TableHead>时间</TableHead><TableHead>记录</TableHead></TableRow></TableHeader><TableBody>{items.map(item => <TableRow key={item.id}><TableCell><time dateTime={item.createdAt}>{new Date(item.createdAt).toLocaleString()}</time></TableCell><TableCell>{label(item)}</TableCell></TableRow>)}</TableBody></Table> : !syncing && !error && <EmptyState title="暂无维护记录" />}
    <p className="hint">最多显示最近 1000 条可用记录。</p></>;
}
