// Coalesce only requests ready in the same turn. Capabilities are neither
// cached nor reused: each retry passes through a fresh authorization call.
export function rangePermits<T>(authorize: (offsets: number[], signal: AbortSignal) => Promise<T[]>) {
  type Pending = { offset: number; resolve(value: T): void; reject(error: unknown): void };
  const groups = new Map<AbortSignal, Pending[]>();
  return (offset: number, signal: AbortSignal): Promise<T> => new Promise((resolve,reject) => {
    if (signal.aborted) { reject(signal.reason); return; }
    const existing = groups.get(signal);
    const request = {offset,resolve,reject};
    if (existing) { existing.push(request); return; }
    const pending = [request]; groups.set(signal,pending);
    queueMicrotask(() => {
      groups.delete(signal);
      void (async () => {
        try {
          signal.throwIfAborted();
          if (pending.length > 6) throw new Error('无效并发设置');
          const permits = await authorize(pending.map(p=>p.offset),signal);
          signal.throwIfAborted();
          if (permits.length !== pending.length) throw new Error('文件授权无效');
          pending.forEach((p,index)=>p.resolve(permits[index]));
        } catch(error) { pending.forEach(p=>p.reject(error)); }
      })();
    });
  });
}
