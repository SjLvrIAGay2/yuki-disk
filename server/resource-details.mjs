const number = value => typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= Number.MAX_SAFE_INTEGER;
const select = (value, names) => Object.fromEntries(names.filter(name => number(value?.[name])).map(name => [name, value[name]]));
export function normalizeTelemetry(raw, kind) {
  if (raw?.version !== 1 || raw.service !== kind || !['development', 'production'].includes(raw.mode) ||
      typeof raw.boot_id !== 'string' || raw.boot_id.length > 80 || !Number.isFinite(Date.parse(raw.sampled_at))) throw new Error('Invalid telemetry');
  const result = { mode: raw.mode, runtime: typeof raw.runtime === 'string' ? raw.runtime.slice(0, 60) : undefined, boot_id: raw.boot_id, sampled_at: raw.sampled_at,
    process: select(raw.process, ['uptime_seconds', 'cpu_seconds', 'rss_bytes', 'heap_bytes', 'goroutines', 'read_bytes', 'write_bytes']),
    host: select(raw.host, ['cpu_count', 'cpu_total', 'cpu_idle', 'memory_total_bytes', 'memory_available_bytes']) };
  if (kind === 'storage') {
    result.network = select(raw.network, ['received_bytes', 'sent_bytes']);
    const storage = select(raw.storage, ['total_bytes', 'free_bytes', 'available_bytes']);
    if (Object.keys(storage).length === 3 && storage.total_bytes >= storage.free_bytes && storage.free_bytes >= storage.available_bytes && typeof raw.storage.read_only === 'boolean') result.storage = { ...storage, read_only: raw.storage.read_only };
    if (result.storage && ['filesystem', 'container'].includes(raw.storage.mode)) Object.assign(result.storage, { mode: raw.storage.mode }, select(raw.storage, ['configured_bytes', 'stored_bytes', 'allocation_unit_bytes']));
  }
  if (kind === 'maintenance' && ['idle', 'executing', 'disconnected'].includes(raw.maintenance?.state)) result.maintenance = { state: raw.maintenance.state, ...select(raw.maintenance, ['last_contact_seconds', 'completed_tasks', 'failed_tasks']) };
  return result;
}

export function addRates(current, previous) {
  const result = { ...current, process: { ...current.process }, host: { ...current.host }, ...(current.network ? { network: { ...current.network } } : {}) };
  if (!previous || previous.boot_id !== current.boot_id) return result;
  const elapsed = current.process.uptime_seconds - previous.process.uptime_seconds;
  if (!(elapsed >= 0.1 && elapsed <= 120)) return result;
  for (const [source, target, factor] of [['cpu_seconds', 'cpu_percent', 100], ['read_bytes', 'read_bytes_per_second', 1], ['write_bytes', 'write_bytes_per_second', 1]]) {
    const delta = current.process[source] - previous.process[source];
    if (Number.isFinite(delta) && delta >= 0) result.process[target] = delta / elapsed * factor;
  }
  if (result.network) for (const [source, target] of [['received_bytes', 'receive_bytes_per_second'], ['sent_bytes', 'send_bytes_per_second']]) {
    const delta = current.network?.[source] - previous.network?.[source];
    if (Number.isFinite(delta) && delta >= 0) result.network[target] = delta / elapsed;
  }
  const total = current.host.cpu_total - previous.host.cpu_total;
  const idle = current.host.cpu_idle - previous.host.cpu_idle;
  if (Number.isFinite(total) && Number.isFinite(idle) && total > 0 && idle >= 0 && idle <= total) result.host.cpu_percent = (total - idle) / total * 100;
  return result;
}

export function projectInventory(raw, services) {
  if (raw?.version !== 1 || !Array.isArray(raw.nodes)) throw new Error('Invalid inventory');
  const control = select(raw.control, ['blocks', 'versions', 'queued_tasks', 'running_tasks', 'failed_tasks', 'database_connections', 'database_in_use', 'database_limit', 'replication_factor']);
  const nodes = {};
  for (const service of services.filter(service => service.kind === 'storage')) {
    const node = raw.nodes.find(node => node.url === new URL(service.url).origin);
    if (!node || !['hot', 'cold'].includes(node.tier)) continue;
    nodes[service.id] = { tier: node.tier, ...select(node, ['replicas']), last_seen: Number.isFinite(Date.parse(node.last_seen)) ? node.last_seen : null, inventory_at: Number.isFinite(Date.parse(node.inventory_at)) ? node.inventory_at : null };
    if (['ready', 'full', 'read_only', 'unavailable'].includes(node.routing_state)) Object.assign(nodes[service.id], { routing_state: node.routing_state }, select(node, ['read_bandwidth_bps', 'write_bandwidth_bps', 'effective_read_bps', 'effective_write_bps', 'receive_bps', 'send_bps', 'reserved_bytes', 'schedulable_bytes']));
  }
  return { control, nodes };
}
