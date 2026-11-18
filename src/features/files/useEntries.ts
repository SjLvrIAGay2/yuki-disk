import { displayError } from '../../lib/errors';
import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../../api/client';
import type { Config, Entry, Page } from '../../api/protocol';
import { collectPages } from '../../api/pages';
export function useEntries(config: Config, token: string, dir: string, trash: boolean, search: string, enabled = true, complete = false) {
  const [items, setItems] = useState<Entry[]>([]); const [cursor, setCursor] = useState('');
  const [readonly, setReadonly] = useState(false);
  const [loading, setLoading] = useState(false); const [error, setError] = useState('');
  const request = useRef<AbortController | undefined>(undefined);
  const load = useCallback(async (after = '') => {
    if (!enabled) return;
    request.current?.abort(); const controller = new AbortController(); request.current = controller;
    setLoading(true); setError('');
    try {
      const params = new URLSearchParams({ path: dir, trash: String(trash), q: search, cursor: after, limit: '100' });
      const read = async (cursor: string) => { params.set('cursor',cursor); return api<Page<Entry> & { readonly?: boolean }>(config, token, `/v1/browse?${params}`, undefined, controller.signal); };
      let result = await read(after);
      if (complete) {
        const first = result; let count = 0;
        const all = await collectPages(async cursor => { const page = cursor ? await read(cursor) : first; count += page.items.length; if(count>50000)throw new Error('目录项目较多，请搜索后再排序。'); return page; },'文件列表发生变化，请刷新。','目录项目较多，请搜索。');
        result = { ...first, items: all, next_cursor: '' };
      }
      if (controller.signal.aborted) return;
      setReadonly(result.readonly === true);
      setItems(old => after ? [...old, ...result.items] : result.items); setCursor(result.next_cursor);
    } catch (e) { if (!controller.signal.aborted) setError(displayError(e, '文件列表暂时无法加载，请重试。')); }
    finally { if (!controller.signal.aborted) setLoading(false); }
  }, [config, token, dir, trash, search, enabled, complete]);
  useEffect(() => { setItems([]); setCursor(''); setLoading(false); setError(''); setReadonly(false); if (!enabled) return; const timer = setTimeout(() => void load(), search ? 250 : 0); return () => { clearTimeout(timer); request.current?.abort(); }; }, [load, search, enabled]);
  return { items, readonly, loading, error, cursor, refresh: () => void load(), more: () => void load(cursor) };
}
