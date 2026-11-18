// This is the complete anonymous projection. Never spread upstream objects here.
export function publicSummary(health, details, config, now = Date.now()) {
  const stale = !health.checkedAt || health.stale;
  const hosts = new Map(); const volumes = new Map(); let receive = 0; let send = 0;
  let cpuMissing = false; let memoryMissing = false; let capacityMissing = false; let networkMissing = !config.services.some(service => service.kind === 'storage');
  const valid = n => typeof n === 'number' && Number.isFinite(n) && n >= 0;
  for (const service of config.services) {
    const sample = details.services.find(item => item.id === service.id)?.telemetry;
    const fresh = sample && now - Date.parse(sample.sampled_at) >= -5000 && now - Date.parse(sample.sampled_at) < config.intervalMs + 2 * config.timeoutMs + 2000;
    if (!fresh || !service.hostGroup) { cpuMissing = true; memoryMissing = true; }
    else if (!hosts.has(service.hostGroup)) hosts.set(service.hostGroup, sample.host);
    if (service.kind !== 'storage') continue;
    if (!fresh || !service.capacityGroup || !sample.storage) capacityMissing = true;
    else if (!volumes.has(service.capacityGroup)) volumes.set(service.capacityGroup, sample.storage);
    if (!fresh || !valid(sample.network?.receive_bytes_per_second) || !valid(sample.network?.send_bytes_per_second)) networkMissing = true;
    else { receive += sample.network.receive_bytes_per_second; send += sample.network.send_bytes_per_second; }
  }
  let cpu = 0; let cores = 0; let memoryTotal = 0; let memoryFree = 0;
  for (const host of hosts.values()) {
    if (!valid(host.cpu_percent) || !valid(host.cpu_count) || host.cpu_count <= 0) cpuMissing = true;
    else { cpu += Math.min(100, host.cpu_percent) * host.cpu_count; cores += host.cpu_count; }
    if (!valid(host.memory_total_bytes) || !valid(host.memory_available_bytes) || host.memory_available_bytes > host.memory_total_bytes) memoryMissing = true;
    else { memoryTotal += host.memory_total_bytes; memoryFree += host.memory_available_bytes; }
  }
  let total = 0; let available = 0;
  for (const volume of volumes.values()) { total += volume.total_bytes; available += volume.available_bytes; }
  const quantum = 64 * 1024 * 1024;
  const capacity = !stale && !capacityMissing && total >= quantum;
  return { state: stale ? (health.checkedAt ? 'stale' : 'unavailable') : health.state === 'ready' ? 'ready' : 'degraded',
    checkedAt: health.checkedAt, stale, intervalMs: config.intervalMs,
    metrics: {
      capacity_bytes: capacity ? Math.round(total / quantum) * quantum : null,
      available_bytes: capacity ? Math.round(available / quantum) * quantum : null,
      storage_used_percent: capacity ? Math.round((total - available) / total * 100) : null,
      cpu_percent: !stale && !cpuMissing && cores ? Math.round(cpu / cores) : null,
      memory_used_percent: !stale && !memoryMissing && memoryTotal ? Math.round((memoryTotal - memoryFree) / memoryTotal * 100) : null,
      receive_bytes_per_second: !stale && !networkMissing ? Math.round(receive / 65536) * 65536 : null,
      send_bytes_per_second: !stale && !networkMissing ? Math.round(send / 65536) * 65536 : null,
    } };
}
