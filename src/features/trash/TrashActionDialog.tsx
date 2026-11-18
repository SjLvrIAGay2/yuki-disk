import { collectVersions, planVersions } from '../files/versionActions';
import { useEffect, useState } from 'react';
import { Alert, Button, Dialog, Progress } from '../../ui';
import type { Config, Entry } from '../../api/protocol';
import { displayError } from '../../lib/errors';
import { collectTrash, trashRequest, planTrashAction } from './actions';
import { useBatchAction } from '../batches/BatchProvider';

type Props = { config: Config; token: string; kind: 'restore' | 'purge' | 'clear'; entries: Entry[]; onClose(): void; onDone(message: string): void; onChanged(): void };
export function TrashActionDialog({ config, token, kind, entries, onClose, onDone }: Props) {
  const batch = useBatchAction('trash'); const task = batch.task;
  const [error, setError] = useState('');
  const operation = task?.kind ?? kind;
  const title = operation === 'clear' ? '清空回收站' : operation === 'purge' ? '永久删除' : '恢复';
  useEffect(() => {
    if (task) return;
    const controller = new AbortController();
    async function prepare() {
      try {
        const snapshot = kind === 'clear' ? await collectTrash(config, token, controller.signal) : entries;
        if (controller.signal.aborted) return;
        const versions = kind === 'clear' ? await collectVersions(config, token, undefined, controller.signal) : [];
        if (controller.signal.aborted) return;
        batch.prepare({ kind, count: snapshot.length + versions.length,
          batches: [...planTrashAction(snapshot), ...planVersions(versions)].map(item => trashRequest(kind === 'restore' ? 'restore' : 'purge', item)) });
      } catch (e) { if (!controller.signal.aborted) setError(displayError(e, '无法读取回收站，请关闭窗口后重试。')); }
    }
    void prepare();
    return () => controller.abort();
  }, [config, token, kind, entries]);
  useEffect(() => {
    if (task?.done && !task.busy) {
      batch.dismiss(); onDone(task.relocated.length ? `已恢复到：${task.relocated.join('、')}` : `${title}完成。`);
    }
  });
  function close() { if (!task?.started) batch.dismiss(); onClose(); }
  const failure = task?.error !== undefined ? displayError(task.error, '操作未完成，可重试剩余项目。') : error;
  return <Dialog open title={title} onOpenChange={open => { if (!open) close(); }}>
    {!task && !error && <Progress value={null} label="正在读取回收站…" />}
    {task && <>
      <p>{operation === 'clear' ? `将永久删除回收站中的全部 ${task.count} 项及其内容，不受搜索条件影响。` : `已选择 ${task.count} 项，操作包含所选文件的全部版本及文件夹内容。`}</p>
      {operation === 'restore' ? <p>恢复到原位置；原文件夹不可用时，恢复到最近可用的上级目录，同名项目自动编号。</p> : <p>永久删除后无法恢复。确认后新移入回收站的项目不会被清空。</p>}
      {task.started && <Progress max={task.batches.length} value={task.next} label={`已完成 ${task.next} / ${task.batches.length} 批`} />}
      {task.stopped && <p>{task.busy ? '当前一批完成后停止。' : '已停止。继续将处理剩余项目。'}</p>}
      {task.started && <p>已完成的操作不会撤销，关闭窗口后可继续查看进度。</p>}
    </>}
    {failure && <Alert variant="danger" title={failure} />}
    <div className="dialog-actions"><Button variant="ghost" onClick={close}>{task?.started ? '关闭' : '取消'}</Button>
      {task?.busy && <Button variant="ghost" disabled={task.stopped} onClick={batch.stop}>停止后续批次</Button>}
      {batch.canReselect && <Button variant="ghost" onClick={() => { batch.dismiss(); onClose(); }}>重新选择</Button>}
      <Button variant={operation === 'restore' ? 'primary' : 'danger'} disabled={task?.busy || !task?.batches.length} onClick={() => void batch.run()}>{task?.busy ? '处理中…' : task?.error !== undefined ? '重试' : task?.stopped ? '继续' : operation === 'restore' ? '确认' : title}</Button></div>
  </Dialog>;
}
