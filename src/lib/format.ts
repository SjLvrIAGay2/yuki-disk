export function bytes(n: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  let i = 0; while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return `${i ? n.toFixed(1) : n} ${units[i]}`;
}
export const date = (s: string | null) => s ? new Date(s).toLocaleString('zh-CN', { hour12: false }) : '—';
export const joinPath = (dir: string, name: string) => dir ? `${dir}/${name}` : name;
export const parentPath = (path: string) => path.includes('/') ? path.slice(0, path.lastIndexOf('/')) : '';
