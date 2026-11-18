export type Settings = { concurrency: number; adaptiveDownloads: boolean; cache: boolean; connection: 'auto' | 'http'; sort: 'name' | 'size' | 'time'; order: 'asc' | 'desc'; density: 'comfortable' | 'compact'; showPath: boolean; notifyComplete: boolean };
export const defaultSettings: Settings = { concurrency: 3, adaptiveDownloads: true, cache: true, connection: 'auto', sort: 'name', order: 'asc', density: 'comfortable', showPath: true, notifyComplete: true };
export function readSettings(): Settings {
  try { const value = JSON.parse(localStorage.getItem('yuki-transfer-settings') || '{}');
    return { concurrency: Number.isInteger(value.concurrency) && value.concurrency >= 1 && value.concurrency <= 6 ? value.concurrency : 3, adaptiveDownloads: value.adaptiveDownloads !== false, cache: value.cache !== false, connection: value.connection === 'http' ? 'http' : 'auto', sort: ['name','size','time'].includes(value.sort) ? value.sort : 'name', order: value.order === 'desc' ? 'desc' : 'asc', density: value.density === 'compact' ? 'compact' : 'comfortable', showPath: value.showPath !== false, notifyComplete: value.notifyComplete !== false };
  } catch { return { ...defaultSettings }; }
}
export function saveSettings(value: Settings) { try { localStorage.setItem('yuki-transfer-settings', JSON.stringify(value)); } catch { /* Preferences remain valid for this page. */ } }
