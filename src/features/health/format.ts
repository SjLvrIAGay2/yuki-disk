export function bytes(value?: number) {
  if (value === undefined || !Number.isFinite(value)) return '未提供';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']; let unit = 0;
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit++; }
  return `${value.toFixed(unit ? 1 : 0)} ${units[unit]}`;
}
export function duration(value?: number) {
  if (value === undefined) return '未提供';
  const seconds = Math.floor(value);
  return `${Math.floor(seconds / 86400)} 天 ${Math.floor(seconds % 86400 / 3600)} 小时 ${Math.floor(seconds % 3600 / 60)} 分`;
}
export const percent = (value?: number) => value === undefined ? '等待采样' : `${value.toFixed(1)}%`;
export const rate = (value?: number) => value === undefined ? '未提供或等待采样' : `${bytes(value)}/s`;
export const date = (value: unknown) => typeof value === 'string' && Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString() : '未上报';
