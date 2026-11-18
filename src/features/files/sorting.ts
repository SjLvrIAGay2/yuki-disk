import type { Entry } from '../../api/protocol';
import type { Settings } from '../settings/settings';
const names = new Intl.Collator('zh-CN', { numeric: true, sensitivity: 'base' });
export const sortOptions = [{value:'name',label:'名称'},{value:'size',label:'大小'},{value:'time',label:'修改时间'}];
export function sortEntries(items: Entry[], sort: Settings['sort'], order: Settings['order']): Entry[] {
  const timestamp = (e: Entry) => { const n = Date.parse(e.modified_at || e.created_at); return Number.isFinite(n) ? n : 0; };
  return [...items].sort((a,b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
    const compared = sort === 'size' && !a.is_dir ? a.size-b.size : sort === 'time' ? timestamp(a)-timestamp(b) : names.compare(a.name,b.name);
    return (compared || names.compare(a.name,b.name) || a.path.localeCompare(b.path)) * (order === 'asc' ? 1 : -1);
  });
}
