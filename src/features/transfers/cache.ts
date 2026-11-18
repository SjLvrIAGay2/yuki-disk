// Verified bytes live only in the current page's authenticated memory scope.
const limitBytes = 256 * 1024 * 1024;
const chunks = new Map<string, ArrayBuffer>();
let retained = 0;
let epoch = 0;
export function cacheEpoch() { return epoch; }
let migration: Promise<void> | undefined;
export function purgeLegacyCache(): Promise<void> {
  if (typeof indexedDB === 'undefined') return Promise.resolve();
  return migration ??= new Promise<void>((resolve, reject) => {
    const request = indexedDB.deleteDatabase('yukidisk-chunks');
    request.onsuccess = () => resolve();
    request.onerror = request.onblocked = () => { migration = undefined; reject(new Error('请关闭其他旧版网盘页面后重试登录。')); };
  });
}
export async function cachedBlock(hash: string, scope = epoch): Promise<ArrayBuffer | undefined> { return scope === epoch ? chunks.get(hash)?.slice(0) : undefined; }
export async function cacheBlock(hash: string, data: ArrayBuffer, scope = epoch): Promise<void> {
  if (scope !== epoch) return;
  retained -= chunks.get(hash)?.byteLength || 0; chunks.delete(hash);
  if (data.byteLength > limitBytes) return;
  chunks.set(hash, data.slice(0)); retained += data.byteLength;
  while (retained > limitBytes) { const key = chunks.keys().next().value!; retained -= chunks.get(key)!.byteLength; chunks.delete(key); }
}
export async function clearCache(): Promise<void> { epoch++; chunks.clear(); retained = 0; }
