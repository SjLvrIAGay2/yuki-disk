import { controlJSON, HTTPError } from './upstream.mjs';

export async function readEvents(config, url, session, stopSignal) {
  if ([...url.searchParams.keys()].some(key => key !== 'after') || url.searchParams.getAll('after').length > 1) throw new HTTPError(400, 'invalid_cursor');
  const after = url.searchParams.get('after') || '0';
  if (!/^(0|[1-9][0-9]{0,15})$/.test(after) || !Number.isSafeInteger(Number(after))) throw new HTTPError(400, 'invalid_cursor');
  const result = await controlJSON(config, `/v1/events?after=${after}&limit=200`, { headers: { Authorization: `Bearer ${session.token}` } }, stopSignal);
  if (!Array.isArray(result.items) || result.items.length > 200 || !Number.isSafeInteger(result.next_after) || result.next_after < Number(after)) throw new HTTPError(503, 'invalid_response');
  let previous = Number(after);
  const items = result.items.map(item => {
    if (!Number.isSafeInteger(item.id) || item.id <= previous || item.id > result.next_after || typeof item.kind !== 'string' || !Number.isFinite(Date.parse(item.created_at))) throw new HTTPError(503, 'invalid_response');
    previous = item.id;
    return { id: item.id, kind: item.kind.slice(0, 80), createdAt: item.created_at, reason: ['cold_migration', 'hot_migration', 'capacity_rebalance', 'bandwidth_rebalance', 'migration_trim', 'repair'].includes(item.detail?.reason) ? item.detail.reason : undefined };
  });
  return { items, nextAfter: result.next_after, more: items.length === 200 };
}
