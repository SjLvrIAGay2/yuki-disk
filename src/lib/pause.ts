export function pause(ms: number, signal: AbortSignal, missingReason: 'reason' | 'abort-error' = 'reason'): Promise<void> {
  return new Promise((resolve, reject) => {
    const cancel = () => { clearTimeout(timer); signal.removeEventListener('abort', cancel); reject(signal.reason ?? (missingReason === 'abort-error' ? new DOMException('已取消', 'AbortError') : signal.reason)); };
    const timer = setTimeout(() => { signal.removeEventListener('abort', cancel); resolve(); }, ms);
    signal.addEventListener('abort', cancel, { once: true }); if (signal.aborted) cancel();
  });
}
