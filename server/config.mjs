import path from 'node:path';

const loopback = new Set(['127.0.0.1', 'localhost', '[::1]', '::1']);
export function origin(value, development) {
  const url = new URL(value);
  if (value !== url.origin || url.username || url.password || url.search || url.hash ||
      !(url.protocol === 'https:' || development && url.protocol === 'http:' && loopback.has(url.hostname))) {
    throw new Error('Expected a safe origin without path, credentials, query or fragment');
  }
  return value;
}
export function prefix(value) {
  if (typeof value !== 'string' || value !== '' && !/^\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*$/.test(value) || value.length > 200) throw new Error('Invalid path prefix');
  return value;
}
export function address(value, development) {
  const url = new URL(value);
  origin(url.origin, development);
  const base = prefix(url.pathname === '/' ? '' : url.pathname);
  if (value !== url.origin + base || url.username || url.password || url.search || url.hash) throw new Error('Invalid service base address');
  return value;
}
export function validateConfig(raw) {
  const allowed = ['host', 'port', 'development', 'publicOrigin', 'publicPath', 'directory', 'cert', 'key', 'control', 'services', 'intervalMs', 'timeoutMs', 'logKey'];
  if (!raw || Object.keys(raw).some(key => !allowed.includes(key))) throw new Error('Unknown config field');
  if(raw.logKey!==undefined&&(typeof raw.logKey!=='string'||raw.logKey.length<32||raw.services?.some(v=>v.monitorKey===raw.logKey)||!raw.development&&raw.logKey.toLowerCase().includes('development')))throw new Error('Invalid log credential');
  const config = { intervalMs: 10000, timeoutMs: 3000, publicPath: '', ...raw };
  if (typeof config.development !== 'boolean' || typeof config.host !== 'string' || !config.host ||
      !Number.isInteger(config.port) || config.port < 1 || config.port > 65535) throw new Error('Invalid listener');
  if (config.development && !loopback.has(config.host)) throw new Error('Development must bind loopback');
  if (!config.development && (!config.cert || !config.key)) throw new Error('Production requires TLS certificate and key');
  origin(config.publicOrigin, config.development); address(config.control, config.development); prefix(config.publicPath);
  if (!Number.isInteger(config.intervalMs) || config.intervalMs < 1000 || config.intervalMs > 60000 ||
      !Number.isInteger(config.timeoutMs) || config.timeoutMs < 100 || config.timeoutMs > 10000) throw new Error('Invalid probe timing');
  if (!Array.isArray(config.services) || !config.services.length || config.services.length > 32) throw new Error('Configure 1..32 services');
  const ids = new Set();
  for (const service of config.services) {
    if (Object.keys(service).some(key => !['id', 'name', 'url', 'kind', 'hostGroup', 'capacityGroup', 'monitorKey'].includes(key)) || !['control', 'storage', 'maintenance', 'web'].includes(service.kind) ||
        !/^[a-z0-9-]{1,40}$/.test(service.id) || ids.has(service.id) ||
        typeof service.name !== 'string' || !service.name.trim() || service.name.length > 80 || /[\x00-\x1f\x7f]/.test(service.name)) throw new Error('Invalid service');
    for (const field of ['hostGroup', 'capacityGroup']) if (service[field] !== undefined && !/^[a-zA-Z0-9_-]{1,80}$/.test(service[field])) throw new Error('Invalid sampling group');
    if (service.monitorKey !== undefined && (typeof service.monitorKey !== 'string' || service.monitorKey.length < 32)) throw new Error('Invalid monitoring credential');
    ids.add(service.id);
    const url = new URL(service.url);
    origin(url.origin, config.development);
    if (!url.pathname.endsWith('/readyz') || url.search || url.hash || url.username || url.password) throw new Error('Only exact readyz targets are allowed');
    address(service.url.slice(0, -'/readyz'.length), config.development);
  }
  if (typeof config.directory !== 'string' || !config.directory || config.directory.includes('\0')) throw new Error('Invalid static directory');
  return { ...config, directory: path.resolve(config.directory) };
}
