// Ordered delivery with a bounded sliding window. A slow block cannot allow
// unbounded read-ahead; completed later blocks still fit in the same window.
export async function orderedPipeline<T>(count: number, window: number | { readonly concurrency: number }, signal: AbortSignal,
  read: (index: number, signal: AbortSignal) => Promise<T>, write: (value: T) => Promise<void>): Promise<void> {
  const stop = new AbortController(); const combined = AbortSignal.any([signal, stop.signal]);
  const pending = new Map<number, Promise<{ value: T } | { error: unknown }>>();
  let next = 0;
  function fill(index: number) {
    const concurrency = typeof window === 'number' ? window : window.concurrency;
    if (!Number.isInteger(concurrency) || concurrency < 1 || concurrency > 6) throw new Error('无效并发设置');
    while (next < count && next < index + concurrency) {
      const current = next++; pending.set(current, read(current, combined).then(value => ({ value }), error => ({ error })));
    }
  }
  try {
    fill(0);
    for (let index = 0; index < count; index++) {
      combined.throwIfAborted(); const result = await pending.get(index)!; pending.delete(index);
      if ('error' in result) throw result.error;
      combined.throwIfAborted(); await write(result.value); fill(index + 1);
    }
  } finally { stop.abort(); await Promise.all(pending.values()); }
}
