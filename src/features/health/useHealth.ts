import { useEffect, useState } from 'react';
import { request } from '../../api/client';
import type { Snapshot } from '../../api/protocol';

export function useHealth() {
  const [snapshot, setSnapshot] = useState<Snapshot>(); const [error, setError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController(); let timer: ReturnType<typeof setTimeout>; let delay = 1000;
    async function load() {
      try {
        const result = await request<Snapshot>('/api/status', { signal: controller.signal });
        if (!controller.signal.aborted) { setSnapshot(result); setError(false); delay = result.checkedAt ? result.intervalMs : 1000; }
      } catch { if (!controller.signal.aborted) setError(true); }
      finally { if (!controller.signal.aborted) timer = setTimeout(load, delay); }
    }
    void load(); return () => { controller.abort(); clearTimeout(timer); };
  }, [attempt]);
  return { snapshot, error, refresh: () => setAttempt(value => value + 1) };
}
