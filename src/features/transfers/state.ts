export type Transfer = { id: string; name: string; progress: number; size?: number; transferred?: number; speed?: number; remaining?: number; completedAt?: string; label: string; state: 'queued' | 'running' | 'done' | 'failed' | 'cancelled'; direction?: 'upload' | 'download' };
type Action = { type: 'add'; item: Transfer } | { type: 'update'; id: string; changes: Partial<Transfer> } | { type: 'clear'; ids?: string[] };
export function transferReducer(state: Transfer[], action: Action): Transfer[] {
  if (action.type === 'add') return [...state, action.item];
  if (action.type === 'clear') return state.filter(t => t.state === 'running' || t.state === 'queued' || !!action.ids && !action.ids.includes(t.id));
  return state.map(t => t.id === action.id ? { ...t, ...action.changes } : t);
}
