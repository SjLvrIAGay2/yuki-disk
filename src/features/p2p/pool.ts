import type { Connection } from './connection';

type Entry = { key: string; session: string; busy: boolean; capacity: number; life: AbortController; connection?: Connection; uses: number; idle?: ReturnType<typeof setTimeout> };
const entries = new Set<Entry>();
let epoch = 0;
const waiters = new Set<() => void>();
const notify = () => { for (const wake of [...waiters]) wake(); };
function remove(entry: Entry) { clearTimeout(entry.idle); entries.delete(entry); entry.life.abort(); entry.connection?.close(); notify(); }
function changed(signal: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    const cleanup = () => { waiters.delete(wake); signal.removeEventListener('abort', abort); };
    const wake = () => { cleanup(); resolve(); }; const abort = () => { cleanup(); reject(signal.reason); };
    waiters.add(wake); signal.addEventListener('abort', abort, { once: true }); if (signal.aborted) abort();
  });
}

// Three concurrent leases for uploads, up to six on download demand; twelve
// connections per tab. A lower-budget caller cannot reuse extra idle channels
// to exceed its active-lease limit after a download has expanded the pool.
// Checkout is exclusive; a failed/cancelled request never returns a dirty channel.
export async function acquire(key: string, session: string, signal: AbortSignal, create: (life: AbortSignal) => Promise<Connection>, capacity = 3) {
  if(!Number.isInteger(capacity)||capacity<1||capacity>6)throw new Error('无效连接限制');
  const stamp = epoch;
  let entry: Entry;
  for (;;) {
    signal.throwIfAborted();
    if (stamp !== epoch) throw new Error("连接已关闭");
    for (const item of entries) if (!item.busy && (!item.connection?.healthy() || item.connection.expires <= Date.now() || item.uses >= (item.connection?.maxRequests ?? 500))) remove(item);
    const matching = [...entries].filter(e => e.key === key);
    if(matching.filter(e=>e.busy&&e.capacity===capacity).length>=capacity || matching.filter(e=>e.busy).length>=6){await changed(signal);continue;}
    const idle = matching.find(e => !e.busy);
    if (idle) { entry = idle; clearTimeout(entry.idle); entry.busy = true;entry.capacity=capacity; break; }
    if (matching.length < 6) {
      if (entries.size >= 12) { const other = [...entries].find(e => !e.busy); if (other) remove(other); }
      if (entries.size < 12) {
        entry = { key, session, busy: true, capacity, life: new AbortController(), uses: 0 }; entries.add(entry);
        try { entry.connection = await create(entry.life.signal); signal.throwIfAborted(); } catch (error) { remove(entry); throw error; }
        break;
      }
    }
    await changed(signal);
  }
  if (!entries.has(entry) || stamp !== epoch) { remove(entry); throw new Error("连接已关闭"); }
  const current = entry; current.uses++;
  let released = false;
  const abort = () => remove(current);
  signal.addEventListener('abort', abort, { once: true }); if (signal.aborted) abort();
  return { connection: current.connection!, release(clean: boolean) {
    if (released) return; released = true; signal.removeEventListener('abort', abort);
    if (!clean || signal.aborted || !entries.has(current)) remove(current);
    else { current.busy = false; scheduleKeepAlive(current); notify(); }
  } };
}
// Heartbeats borrow the same exclusive channel as requests. A missing pong
// retires it; no background reconnect or ticket loop runs when the user is idle.
function scheduleKeepAlive(entry: Entry) {
  entry.idle = setTimeout(async () => {
    if (!entries.has(entry)) return;
    if (!entry.connection?.healthy() || entry.connection.expires <= Date.now()) { remove(entry); return; }
    if (!entry.connection.keepAlive) { remove(entry); return; }
    entry.busy = true;
    const deadline = setTimeout(() => remove(entry), 10000);
    try { await entry.connection.keepAlive(); }
    catch { remove(entry); }
    finally { clearTimeout(deadline); }
    if (entries.has(entry)) { entry.busy = false; scheduleKeepAlive(entry); notify(); }
  }, 60000);
}

export function closeConnections(session?: string) { epoch++; notify(); for (const e of entries) if (!session || e.session === session) remove(e); }
if (typeof window !== 'undefined') window.addEventListener('pagehide', () => closeConnections());
