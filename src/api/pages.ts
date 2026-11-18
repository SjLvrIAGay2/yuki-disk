import type { Page } from './protocol';

export async function collectPages<T>(read: (cursor: string) => Promise<Page<T>>, repeated: string, excessive: string): Promise<T[]> {
  const items: T[] = []; const seen = new Set<string>(); let cursor = '';
  do {
    const page = await read(cursor); items.push(...page.items); cursor = page.next_cursor;
    if (cursor && seen.has(cursor)) throw new Error(repeated);
    seen.add(cursor);
    if (seen.size > 10000) throw new Error(excessive);
  } while (cursor);
  return items;
}
