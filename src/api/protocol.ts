type Service = { id: string; name: string; state: 'ready' | 'unavailable'; latencyMs: number };
export type Snapshot = { state: 'checking' | 'ready' | 'degraded' | 'stale' | 'unavailable'; checkedAt: string | null; stale: boolean; intervalMs: number; services?: Service[]; metrics: Record<string, number | null> };
export type Session = { token: string; expiresAt: number };
export type Event = { id: number; kind: string; reason?: 'cold_migration' | 'hot_migration' | 'capacity_rebalance' | 'bandwidth_rebalance' | 'migration_trim' | 'repair'; createdAt: string };
export type EventPage = { items: Event[]; nextAfter: number; more: boolean };
type Telemetry = {
  mode: 'development' | 'production'; runtime?: string; sampled_at: string;
  process: Partial<Record<'uptime_seconds' | 'cpu_seconds' | 'rss_bytes' | 'heap_bytes' | 'goroutines' | 'cpu_percent' | 'read_bytes_per_second' | 'write_bytes_per_second', number>>;
  host: Partial<Record<'cpu_count' | 'cpu_percent' | 'memory_total_bytes' | 'memory_available_bytes', number>>;
  storage?: { total_bytes: number; free_bytes: number; available_bytes: number; read_only: boolean; mode?: 'filesystem' | 'container'; configured_bytes?: number; stored_bytes?: number; allocation_unit_bytes?: number };
  maintenance?: { state: 'idle' | 'executing' | 'disconnected'; last_contact_seconds?: number; completed_tasks?: number; failed_tasks?: number };
};
export type ServiceDetail = { id: string; name: string; kind: string; state: 'ready' | 'unavailable'; telemetry?: Telemetry; inventory?: { tier?: 'hot' | 'cold'; last_seen?: string | null; inventory_at?: string | null; [key: string]: unknown } };
export type Details = { checkedAt: string | null; stale: boolean; inventoryState: 'ready' | 'unavailable'; services: ServiceDetail[] };
