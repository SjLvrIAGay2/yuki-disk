import { versionRequest, type VersionBatch } from '../files/versionActions';
import type { BatchRequest } from '../batches/tasks';
import { api } from '../../api/client';
import { collectPages } from '../../api/pages';
import { outerPaths } from '../../lib/selection';
import type { Config, Entry, Page } from '../../api/protocol';

type TrashTarget = { path: string; trash_id: string };
type PathTrashBatch = { items: TrashTarget[]; key: string; versions?: never };
type TrashBatch = PathTrashBatch | (VersionBatch & { items?: never });
export function planTrashAction(entries: Entry[]): PathTrashBatch[] {
  const targets = entries.map(entry => {
    if (!entry.trash_id) throw new Error('回收站列表已更新，请刷新。');
    return { path: entry.path, trash_id: entry.trash_id };
  });
  const roots = outerPaths(targets, item => item.path);
  const batches: PathTrashBatch[] = [];
  for (let i = 0; i < roots.length; i += 100) batches.push({ items: roots.slice(i, i + 100), key: crypto.randomUUID() });
  return batches;
}

// A clear confirmation freezes all pages, independently of the current search and loaded page.
export async function collectTrash(config: Config, token: string, signal: AbortSignal): Promise<Entry[]> {
  return collectPages(cursor => api<Page<Entry>>(config, token, `/v1/browse?${new URLSearchParams({ trash: 'true', limit: '200', cursor })}`, undefined, signal),
    '回收站分页异常，请刷新。', '回收站项目过多，请分批处理。');
}

export function trashRequest(operation: 'restore' | 'purge', batch: TrashBatch): BatchRequest {
  if (batch.versions) return versionRequest(operation, batch);
  return { route: '/v1/trash/actions', body: { operation, items: batch.items }, key: batch.key };
}
