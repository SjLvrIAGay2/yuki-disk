import { api } from '../../api/client';
import type { Config } from '../../api/protocol';

export type BatchRequest = { route: '/v1/trash/actions' | '/v1/versions/actions' | '/v1/operations'; key: string; body: unknown };
export type BatchIntent = { kind: 'trash' | 'restore' | 'purge' | 'clear' | 'move' | 'copy' | 'rename'; count: number; batches: BatchRequest[] };
type BatchTask = BatchIntent & { next: number; busy: boolean; stopped: boolean; started: boolean; done: boolean; error?: unknown; relocated: string[] };
type Result = { items?: { path?: string; target?: string }[] };

// One registry per logged-in workspace. Navigation never discards retry identities.
// Stop waits for the current atomic request; an uncertain result keeps its key.
export class BatchTasks {
  private tasks = new Map<string, BatchTask>();
  private listeners = new Set<() => void>();
  private closed = false;
  constructor(private execute: (batch: BatchRequest) => Promise<Result>) {}
  snapshot = () => this.tasks;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private update(scope: string, task: BatchTask) {
    this.tasks = new Map(this.tasks).set(scope, task);
    for (const listener of this.listeners) listener();
  }
  prepare(scope: string, intent: BatchIntent) {
    if (this.closed || this.tasks.has(scope)) return;
    this.update(scope, { ...intent, next: 0, busy: false, stopped: false, started: false, done: false, relocated: [] });
  }
  dismiss(scope: string) {
    const task = this.tasks.get(scope);
    if (task?.busy) return;
    this.tasks = new Map(this.tasks); this.tasks.delete(scope);
    for (const listener of this.listeners) listener();
  }
  stop(scope: string) {
    const task = this.tasks.get(scope);
    if (task && !task.done) this.update(scope, { ...task, stopped: true });
  }
  async run(scope: string) {
    let task = this.tasks.get(scope);
    if (!task || task.busy || task.done || this.closed) return;
    this.update(scope, { ...task, busy: true, started: true, stopped: false, error: undefined });
    try {
      while ((task = this.tasks.get(scope)!) && !task.stopped && !this.closed && task.next < task.batches.length) {
        const result = await this.execute(task.batches[task.next]);
        const latest = this.tasks.get(scope)!;
        const next = latest.next + 1;
        this.update(scope, { ...latest, next, done: next === latest.batches.length,
          relocated: [...latest.relocated, ...(result.items ?? []).filter(item => item.target && item.target !== item.path).map(item => item.target!)] });
      }
    } catch (error) {
      this.update(scope, { ...this.tasks.get(scope)!, error });
    } finally {
      this.update(scope, { ...this.tasks.get(scope)!, busy: false });
    }
  }
  close() { this.closed = true; for (const scope of this.tasks.keys()) this.stop(scope); }
}

export function createBatchTasks(config: Config, token: string) {
  return new BatchTasks(batch => api<Result>(config, token, batch.route, batch.body, undefined, batch.key));
}
