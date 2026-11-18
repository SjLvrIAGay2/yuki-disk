import { api } from '../../api/client';
import { collectPages } from '../../api/pages';
import type { Config, Page, Version } from '../../api/protocol';
import type { BatchRequest } from '../batches/tasks';
export type VersionBatch = { versions: { id: string; trash_id?: string }[]; key: string };
export async function collectVersions(config: Config, token: string, path: string | undefined, signal: AbortSignal) {
  const items = await collectPages(async cursor => {
    const query = new URLSearchParams({ limit: '200', cursor }); if (path === undefined) query.set('trash', 'true'); else query.set('path', path);
    return api<Page<Version>>(config, token, `/v1/versions?${query}`, undefined, signal);
  }, '版本列表已变化，请刷新。', '版本过多，请分批处理。');
  return items.sort((a, b) => b.created_at.localeCompare(a.created_at));
}
export function planVersions(items: Version[]): VersionBatch[] {
  const batches: VersionBatch[] = [];
  for (let i = 0; i < items.length; i += 100) batches.push({ key: crypto.randomUUID(), versions: items.slice(i, i + 100).map(v => ({ id: v.id, ...(v.trash_id ? { trash_id: v.trash_id } : {}) })) });
  return batches;
}
export function versionRequest(operation: 'trash' | 'restore' | 'purge', batch: VersionBatch): BatchRequest {
  return { route: '/v1/versions/actions', body: { operation, items: batch.versions }, key: batch.key };
}
