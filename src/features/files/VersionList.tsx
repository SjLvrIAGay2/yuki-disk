import { select } from '../../lib/selection';
import { FileVersionCards } from './FileVersionCards';
import { useEffect, useState } from 'react';
import { Alert, Button, Checkbox, Progress, Badge } from '../../ui';
import type { Config, Version } from '../../api/protocol';
import { bytes, date } from '../../lib/format';
import { displayError } from '../../lib/errors';
import { collectVersions, planVersions, versionRequest } from './versionActions';
import { useBatchAction } from '../batches/BatchProvider';
export function VersionList({ config, token, path, refresh = 0, onChanged, save, preview }: { config: Config; token: string; path?: string; refresh?: number | string; preview?(version: Version): void; onChanged(): void; save(id: string, name: string): void }) {
 const [items, setItems] = useState<Version[]>([]); const [selected, setSelected] = useState(new Set<string>());
 const [error, setError] = useState(''); const [loading, setLoading] = useState(true); const [revision, setRevision] = useState(0);
 const batch = useBatchAction('versions:' + (path ?? '__trash'));
 const action = batch.task; const busy = action?.busy;
 useEffect(() => { if (action?.done && !action.busy) { batch.dismiss(); setRevision(v => v + 1); onChanged(); } });
 useEffect(() => { const controller = new AbortController(); setLoading(true); setError('');
   void collectVersions(config, token, path, controller.signal).then(all => { if (!controller.signal.aborted) { setItems(preview ? all.filter(v => !v.deleted) : all); setSelected(new Set()); } }).catch(e => { if (!controller.signal.aborted) setError(displayError(e, '无法加载版本，请重试。')); }).finally(() => { if (!controller.signal.aborted) setLoading(false); });
   return () => controller.abort();
 }, [config, token, path, refresh, revision, !!preview]);
 const chosen = items.filter(v => selected.has(v.id));
 function prepare(kind: 'trash' | 'restore' | 'purge', selection = chosen) { setError(''); batch.prepare({ kind, batches: planVersions(selection).map(b => versionRequest(kind, b)), count: selection.length }); }
 if (action) return <div><p>{action.kind === 'trash' ? `将选中的 ${action.count} 个版本移入回收站，其他版本不受影响。` : action.kind === 'restore' ? `恢复选中的 ${action.count} 个版本。` : `永久删除选中的 ${action.count} 个版本，删除后无法恢复。`}</p>
   {action.started && <Progress max={action.batches.length} value={action.next} label={`已完成 ${action.next} / ${action.batches.length} 批`} />}
   {action.stopped && <p>{busy ? '当前一批完成后停止。' : '已停止。继续将处理剩余版本。'}</p>}
   {action.error !== undefined && <Alert variant="danger" title={displayError(action.error, '操作未完成，可重试剩余版本。')} />}
   <div className="dialog-actions">{!action.started ? <Button variant="ghost" onClick={batch.dismiss}>取消</Button> : <Button variant="ghost" disabled={!busy || action.stopped} onClick={batch.stop}>停止后续批次</Button>}
   {batch.canReselect && <Button variant="ghost" onClick={() => { batch.dismiss(); setRevision(v => v + 1); onChanged(); }}>重新选择</Button>}
   <Button variant={action.kind === 'restore' ? 'primary' : 'danger'} disabled={busy} onClick={() => void batch.run()}>{busy ? '处理中…' : action.error !== undefined ? '重试' : action.stopped ? '继续' : '确认'}</Button></div></div>;
 return <div>{error && <Alert variant="danger" title={error} />}
 <div className="batch-toolbar"><span>已选择 {chosen.length} 个版本</span><Button variant="ghost" disabled={!chosen.length || chosen.some(v => v.trashed_at)} onClick={() => prepare('trash')}>删除所选版本</Button>{!preview && <><Button variant="ghost" disabled={!chosen.length || chosen.some(v => !v.deleted)} onClick={() => prepare('restore')}>恢复版本</Button><Button variant="ghost" disabled={!chosen.length || chosen.some(v => !v.deleted)} onClick={() => prepare('purge')}>永久删除版本</Button></>}<Button variant="ghost" disabled={loading} onClick={() => setRevision(v => v + 1)}>刷新版本</Button></div>
 {loading && <Progress value={null} label="加载中…" />}
 <>{preview ? <FileVersionCards items={items} selected={selected} select={(id, checked) => setSelected(old => select(old, id, checked))} preview={preview} save={save} remove={v => prepare('trash', [v])} /> : <div className="version-list">{items.map(v => <article key={v.id}><Checkbox label={`选择 ${v.path} ${date(v.created_at)}`} checked={selected.has(v.id)} onCheckedChange={value => setSelected(old => select(old, v.id, value === true))} /><div>{path === undefined && <strong>{v.path}</strong>}<strong>{date(v.created_at)}</strong><small>{bytes(v.size)}</small>{v.trashed_at && <><Badge>已删除</Badge><small>可恢复至 {date(v.expires_at!)}</small></>}</div><Button variant="ghost" onClick={() => save(v.id, v.path.split('/').pop()!)}>下载此版本</Button></article>)}</div>}</>{!loading && !items.length && <p>暂无版本。</p>}
 </div>;
}
