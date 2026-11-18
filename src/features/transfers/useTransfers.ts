import { useToast } from '../../ui';
import { displayError } from '../../lib/errors';
import { useEffect, useReducer, useRef } from 'react';
import type { Config } from '../../api/protocol';
import { APIError } from '../../api/client';
import type { Settings } from '../settings/settings';
import { Queue } from './queue';
import { transferReducer } from './state';
import { upload } from './upload';
import { downloadTo } from './download';
import { openDestination } from './save';
import { TransferMeter, type TransferProgress } from './metrics';

type Job = { name: string; path?: string; file?: File; version?: string; idem: string; controller?: AbortController };
export function useTransfers(config: Config, token: string, settings: Settings, onUpload: () => void) {
  const { toast } = useToast(); const alive = useRef(true);
  const [transfers, dispatch] = useReducer(transferReducer, []);
  const queue = useRef(new Queue(2)); const jobs = useRef(new Map<string, Job>());
  const current = useRef({ settings, onUpload }); current.current = { settings, onUpload };
  useEffect(() => { alive.current = true; return () => { alive.current = false; for (const job of jobs.current.values()) job.controller?.abort(); }; }, []);
  function run(id: string, job: Job) {
    const controller = new AbortController(); job.controller = controller;
    dispatch({ type: 'update', id, changes: { state: 'queued', progress: 0, speed:undefined,remaining:undefined,transferred:0,completedAt: undefined, label: '等待传输' } });
    queue.current.add(async () => {
      const meter=new TransferMeter();let tick:ReturnType<typeof setInterval>|undefined;let last=0;
      const progress: TransferProgress = (progress, label, detail) => {if(detail)meter.update(detail);const now=performance.now();if(progress<100&&detail?.phase==='transfer'&&now-last<100)return;last=now;dispatch({ type: 'update', id, changes: { progress, label,...meter.snapshot() } });};
      try {
        controller.signal.throwIfAborted(); dispatch({ type: 'update', id, changes: { state: 'running' } });
        tick=setInterval(()=>dispatch({type:'update',id,changes:meter.snapshot()}),1000);
        let size = job.file?.size;
        const preferences = current.current.settings;
        const transferConfig = { ...config, connection: preferences.connection };
        if (job.file) {
          await upload(transferConfig, token, job.file, job.path!, controller.signal, progress, preferences.concurrency, job.idem);
          current.current.onUpload();
        } else {
          const destination = await openDestination(job.name);
          size = 0;
          const sink = { write: async (data: ArrayBuffer) => { await destination.sink.write(data); size! += data.byteLength; }, close: () => destination.sink.close(), abort: () => destination.sink.abort() };
          await downloadTo(transferConfig, token, job.version!, controller.signal, sink, progress, preferences.concurrency, preferences.cache, destination.maxSize, undefined, undefined, preferences.adaptiveDownloads);
        }
        const label = job.file ? '上传完成' : '下载完成';
        dispatch({ type: 'update', id, changes: { state: 'done', progress: 100, size, speed:undefined,remaining:undefined,completedAt: new Date().toISOString(), label } });
        if (alive.current && current.current.settings.notifyComplete) toast({ title: label, description: job.path || job.name, variant: 'success' });
      } catch (e) {
        if (e instanceof APIError && e.code === 'upload_expired') job.idem = crypto.randomUUID();
        const cancelled = controller.signal.aborted;
        const label = cancelled ? '已取消' : displayError(e, '传输未完成，请重试。');
        dispatch({ type: 'update', id, changes: { state: cancelled ? 'cancelled' : 'failed', label, speed:undefined,remaining:undefined } });
        if (alive.current) toast({ title: `${job.file ? '上传' : '下载'}${cancelled ? '已取消' : '失败'}`, description: `${job.path || job.name}${cancelled ? '' : `：${label}`}`, variant: cancelled ? 'info' : 'danger' });
      } finally { clearInterval(tick);if (job.controller === controller) job.controller = undefined; }
    });
  }
  function add(job: Job) {
    const id = crypto.randomUUID(); jobs.current.set(id, job);
    dispatch({ type: 'add', item: { id, name: job.path || job.name, progress: 0, size: job.file?.size, label: '等待传输', state: 'queued', direction: job.file ? 'upload' : 'download' } });
    run(id, job);
  }
  function enqueue(files: FileList | File[], dir: string) {
    for (const file of Array.from(files)) {
      const relative = file.webkitRelativePath || file.name;
      add({ name: file.name, path: dir ? `${dir}/${relative}` : relative, file, idem: crypto.randomUUID() });
    }
  }
  async function save(version: string, name: string) {
    add({ name, version, idem: crypto.randomUUID() });
  }
  function retry(id: string) {
    const job = jobs.current.get(id); if (!job || job.controller) return;
    // Retain the reservation owner and commit identity when resuming this job.
    run(id, job);
  }
  function clear(ids?: string[]) {
    for (const [id, job] of jobs.current) if (!job.controller && (!ids || ids.includes(id))) jobs.current.delete(id);
    dispatch({ type: 'clear', ids });
  }
  return { transfers, enqueue, save, retry, clear, cancel: (id: string) => jobs.current.get(id)?.controller?.abort() };
}
