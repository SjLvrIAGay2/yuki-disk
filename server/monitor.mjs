import { readJSON } from './upstream.mjs';
import { normalizeTelemetry, addRates } from './resource-details.mjs';
import { serviceTelemetry } from './telemetry.mjs';

export function createMonitor(config, stopSignal, log=()=>{}) {
  let snapshot = { state: 'checking', checkedAt: null, intervalMs: config.intervalMs, services: [] };
  let pending; let timer; let closed = false;
  let resources = []; const previous = new Map();
  async function inspect(service, read) {
    const start = performance.now();
    const base = { id: service.id, name: service.name };
    let raw; let ready = false; let detail = { ...base, kind: service.kind, state: 'unavailable' };
    try {
      raw = await read(service, new URL('statusz', service.url).href, 32768);
      const sample = normalizeTelemetry(raw, service.kind);
      const telemetry = addRates(sample, previous.get(service.id)); previous.set(service.id, sample);
      detail = { ...detail, state: 'ready', telemetry };
    } catch { previous.delete(service.id); }
    // Readiness is independent of optional telemetry. Old or failed statusz
    // responses still use the original bounded readiness contract.
    if (raw?.version === 1 && raw.service === service.kind && typeof raw.ready === 'boolean') ready = raw.ready;
    else {
      try {
        const body = await read(service, service.url, 4096);
        ready = service.kind === 'storage' ? body.ready === true : body.status === 'ready';
      } catch { /* The service remains unavailable; monitoring never blocks startup. */ }
    }
    return { health: { ...base, state: ready ? 'ready' : 'unavailable', latencyMs: Math.round(performance.now() - start) }, detail };
  }
  function refresh() {
    if (closed) return Promise.resolve();
    if (pending) return pending;
    pending = (async () => {
      // Only share requests inside this round, never cache a health result.
      const requests = new Map();
      function read(service, url, maxBytes) {
        const requestKey = JSON.stringify([url, service.monitorKey || '']);
        if (!requests.has(requestKey)) requests.set(requestKey, (async () => {
          const response = await fetch(url, { headers: service?.monitorKey ? { 'X-Yuki-Monitor': service.monitorKey } : {}, redirect: 'error', signal: AbortSignal.any([stopSignal, AbortSignal.timeout(config.timeoutMs)]) });
          if (!response.ok) { await response.body?.cancel(); throw new Error('Probe unavailable'); }
          return readJSON(response, maxBytes);
        })());
        return requests.get(requestKey);
      }
      const results = await Promise.all(config.services.map(service => inspect(service, read)));
      const services = results.map(result => result.health); const details = results.map(result => result.detail);
      const self = normalizeTelemetry(serviceTelemetry('status', config.development), 'status');
      resources = [...details, { id: '__status', name: '状态服务', kind: 'status', state: 'ready', telemetry: addRates(self, previous.get('__status')) }]; previous.set('__status', self);
      const nextState=services.every(service=>service.state==='ready')?'ready':'degraded';
      if(nextState!==snapshot.state){if(nextState==='degraded')log('warn','probe_failed');else if(snapshot.state!=='checking')log('info','probe_recovered')}
      snapshot = { state: services.every(service => service.state === 'ready') ? 'ready' : 'degraded', checkedAt: new Date().toISOString(), intervalMs: config.intervalMs, services };
    })().finally(() => { pending = undefined; if (!closed) timer = setTimeout(refresh, config.intervalMs).unref(); });
    return pending;
  }
  void refresh();
  return {
    details: () => ({ checkedAt: snapshot.checkedAt, services: resources }),
    snapshot: () => ({ ...snapshot, stale: !snapshot.checkedAt || Date.now() - Date.parse(snapshot.checkedAt) > config.intervalMs + 2 * config.timeoutMs + 2000 }),
    async close() { closed = true; clearTimeout(timer); await pending; },
  };
}
