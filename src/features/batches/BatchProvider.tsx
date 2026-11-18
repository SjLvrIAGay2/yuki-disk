import { createContext, useContext, useEffect, useState, useSyncExternalStore, type ReactNode } from 'react';
import type { Config } from '../../api/protocol';
import { APIError } from '../../api/client';
import { createBatchTasks, type BatchIntent, type BatchTasks } from './tasks';

const Context = createContext<BatchTasks | undefined>(undefined);
export function BatchProvider({ config, token, children }: { config: Config; token: string; children: ReactNode }) {
  const [tasks] = useState(() => createBatchTasks(config, token));
  useEffect(() => () => tasks.close(), [tasks]);
  return <Context.Provider value={tasks}>{children}</Context.Provider>;
}
export function useBatchAction(scope: string) {
  const tasks = useContext(Context);
  if (!tasks) throw new Error('Batch workspace is unavailable');
  const all = useSyncExternalStore(tasks.subscribe, tasks.snapshot);
  const task = all.get(scope);
  // A received client-error response rejects the atomic batch. A lost response
  // or server failure cannot justify replacing its original retry identity.
  const canReselect = !task?.busy && task?.error instanceof APIError && task.error.status >= 400 && task.error.status < 500;
  return { task, canReselect, prepare: (intent: BatchIntent) => tasks.prepare(scope, intent),
    run: () => tasks.run(scope), stop: () => tasks.stop(scope), dismiss: () => tasks.dismiss(scope) };
}
