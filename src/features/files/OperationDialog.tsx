import { Button, Input, Dialog, Alert, Progress } from '../../ui';
import { displayError } from '../../lib/errors';
import { useEffect, useState, type FormEvent } from 'react';
import { api } from '../../api/client';
import type { Config } from '../../api/protocol';
import { joinPath, parentPath } from '../../lib/format';
import { outerPaths } from '../../lib/selection';
import { useBatchAction } from '../batches/BatchProvider';
import type { BatchIntent, BatchRequest } from '../batches/tasks';
const titles: Record<string, string> = { mkdir: '新建文件夹', rename: '重命名', move: '移动', copy: '复制', trash: '移入回收站', restore: '恢复' };
type Props = { config: Config; token: string; kind: string; paths: string[]; dir: string; onClose(): void; onDone(): void };
export function OperationDialog({ config, token, kind, paths, dir, onClose, onDone }: Props) {
  const batch = useBatchAction('files'); const task = batch.task;
  const [value, setValue] = useState(() => {
    const frozen = (task?.batches[0]?.body as { target?: string } | undefined)?.target;
    return kind === 'rename' ? (frozen ?? paths[0]).split('/').pop()! : ['move', 'copy'].includes(kind) ? frozen ?? dir : '';
  });
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [key] = useState(crypto.randomUUID());
  useEffect(() => { if (task?.done && !task.busy) { batch.dismiss(); onDone(); } });
  function close() { if (!task?.started) batch.dismiss(); onClose(); }
  async function submit(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError('');
    try {
      if ((kind === 'mkdir' || kind === 'rename') && (!value || value.includes('/'))) throw new Error('名称不能为空或包含斜杠');
      if (kind === 'mkdir') await api(config, token, '/v1/folders', { path: joinPath(dir, value) });
      else {
        if (!task) {
          const roots = outerPaths(paths, path => path); const batches: BatchRequest[] = [];
          for (let index = 0; index < roots.length; index += 100) batches.push({ route: '/v1/operations', key: index === 0 ? key : crypto.randomUUID(),
            body: { operation: kind, paths: roots.slice(index, index + 100), target: kind === 'rename' ? joinPath(parentPath(paths[0]), value) : ['move', 'copy'].includes(kind) ? value : '' } });
          batch.prepare({ kind: kind as BatchIntent['kind'], count: roots.length, batches });
        }
        await batch.run();
      }
      if (kind === 'mkdir') onDone();
    } catch (e) { setError(displayError(e, '操作未完成，请重试。')); } finally { setBusy(false); }
  }
  return <Dialog open title={titles[kind]} onOpenChange={open => { if (!open && kind !== 'mkdir') close(); else if (!open && !busy) close(); }}><form onSubmit={submit}>
    {paths.length > 0 && <p className="operation-summary">已选择 {paths.length} 项：{paths.map(p => p.split('/').pop()).join('、')}</p>}
    {['mkdir', 'rename', 'move', 'copy'].includes(kind) && <Input label={['mkdir', 'rename'].includes(kind) ? '名称' : '目标文件夹路径（留空为根目录）'} autoFocus value={value} disabled={task?.started} onChange={e => setValue(e.target.value)} required={['mkdir', 'rename'].includes(kind)} />}
    {kind === 'copy' && <p>复制文件的最新版本及文件夹内容。</p>}
    {kind === 'trash' && <p>所选文件的全部版本及文件夹内容将进入回收站，可在保留期限内恢复。</p>}
    {kind === 'restore' && <p>恢复所选项目及其文件夹内容。</p>}
    {error && <Alert variant="danger" title={error} />}
    {task?.error !== undefined && <Alert variant="danger" title={displayError(task.error, '操作未完成，请重试。')} />}
    {task?.started && <><Progress max={task.batches.length} value={task.next} label={`已完成 ${task.next} / ${task.batches.length} 批`} /><p>{task.stopped ? task.busy ? '当前一批完成后停止。' : '已停止。继续将处理剩余项目。' : '关闭窗口后可继续查看进度。'}已完成的操作不会撤销。</p></>}
    <div className="dialog-actions"><Button variant="ghost" type="button" onClick={close} disabled={kind === 'mkdir' && busy}>{task?.started ? '关闭' : '取消'}</Button>
      {task?.busy && <Button type="button" variant="ghost" disabled={task.stopped} onClick={batch.stop}>停止后续批次</Button>}
      {batch.canReselect && <Button type="button" variant="ghost" onClick={() => { batch.dismiss(); onDone(); }}>重新选择</Button>}
      <Button variant="primary" type="submit" disabled={busy || task?.busy}>{busy || task?.busy ? '处理中…' : task?.error !== undefined ? '重试' : task?.stopped ? '继续' : '确认'}</Button></div>
  </form></Dialog>;
}
