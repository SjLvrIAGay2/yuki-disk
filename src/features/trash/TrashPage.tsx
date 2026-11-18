import { select } from '../../lib/selection';
import { VersionList } from '../files/VersionList';
import { useEffect, useState } from 'react';
import { Alert, Badge, Button, Checkbox, Input, Progress } from '../../ui';
import type { Config, Entry } from '../../api/protocol';
import { useEntries } from '../files/useEntries';
import { TrashTable } from './TrashTable';
import { TrashActionDialog } from './TrashActionDialog';
import { useBatchAction } from '../batches/BatchProvider';

export function TrashPage({ config, token, preview, save }: { config: Config; token: string; preview(entry: Entry): void; save(id: string, name: string): void }) {
  const [search, setSearch] = useState(''); const [selected, setSelected] = useState(new Set<string>());
  const [action, setAction] = useState<{ kind: 'restore' | 'purge' | 'clear'; entries: Entry[] }>(); const [message, setMessage] = useState('');
  const [revision, setRevision] = useState(0);
  const pending = useBatchAction('trash').task;
  const list = useEntries(config, token, '', true, search);
  function refresh() { list.refresh(); setRevision(v => v + 1); }
  useEffect(() => setSelected(new Set()), [search]);
  useEffect(() => setSelected(old => new Set([...old].filter(path => list.items.some(item => item.path === path)))), [list.items]);
  const selection = list.items.filter(item => selected.has(item.path));
  function toggle(path: string) { setSelected(old => select(old, path)); }
  return <section>
    <div className="list-toolbar"><h2>已删除项目 <Badge>{list.items.length} 项</Badge></h2><Input aria-label="搜索文件" placeholder="搜索文件或原路径" value={search} onChange={e => setSearch(e.target.value)} /><Button variant="ghost" disabled={list.loading || !!action} onClick={refresh}>刷新</Button><Button variant="outline" disabled={!!action} onClick={() => setAction({ kind: 'clear', entries: [] })}>清空回收站</Button></div>
    {list.error && <Alert variant="danger" title={list.error} />}{message && <Alert title={message} />}
    {pending && !action && <Button variant="outline" onClick={() => setAction({ kind: pending.kind as 'restore' | 'purge' | 'clear', entries: [] })}>{pending.busy ? '查看操作进度' : '查看上次操作'}</Button>}
    <div className="batch-toolbar"><Checkbox label="全选" checked={list.items.length > 0 && selection.length === list.items.length ? true : selection.length ? 'indeterminate' : false} onCheckedChange={checked => setSelected(checked === true ? new Set(list.items.map(item => item.path)) : new Set())} /><span>已选择 {selection.length} 项</span><Button variant="ghost" disabled={!selection.length} onClick={() => setAction({ kind: 'restore', entries: selection })}>恢复</Button><Button variant="ghost" disabled={!selection.length} onClick={() => setAction({ kind: 'purge', entries: selection })}>永久删除</Button></div>
    <TrashTable config={config} token={token} onChanged={refresh} save={save} items={list.items} selected={selected} toggle={toggle} action={(kind, entries) => setAction({ kind, entries })} preview={preview} />
    {list.loading && <Progress value={null} label="加载中…" />}{list.cursor && <Button variant="ghost" disabled={list.loading} onClick={list.more}>加载更多</Button>}
    <h2>已删除版本</h2><VersionList config={config} token={token} refresh={revision} onChanged={list.refresh} save={save} />
    {action && <TrashActionDialog {...action} config={config} token={token} onChanged={refresh} onClose={() => { setAction(undefined); refresh(); }} onDone={result => { setMessage(result); setAction(undefined); setSelected(new Set()); refresh(); }} />}
  </section>;
}
