import type { Config } from '../api/protocol';
import { clearCache, purgeLegacyCache } from '../features/transfers/cache';

export type Identity = { id: string; username: string; role: 'admin' | 'user' };
export type BrowserSession = { token: string; expires_in: number; user: Identity };
let current: BrowserSession | undefined;
let listener: ((session: BrowserSession | undefined) => void) | undefined;
let pending: Promise<BrowserSession> | undefined;
let generation = 0;
let channel: BroadcastChannel | undefined;
export function accessToken() { return current?.token; }
export function sameSession(token: string) {
  try { const id = (raw: string) => JSON.parse(atob(raw.split('.')[0].replace(/-/g, '+').replace(/_/g, '/'))).jti;
    return !!current && id(token) === id(current.token);
  } catch { return false; }
}
function update(session?: BrowserSession) { if (!session || current?.user.id !== session.user.id) void clearCache(); current = session; listener?.(session); }
export function subscribeSession(callback: typeof listener) {
  listener = callback;
  if (typeof BroadcastChannel !== 'undefined') {
    channel = new BroadcastChannel('yuki-session');
    channel.onmessage = event => { if (event.data === 'signed_out') { generation++; update(); } };
  }
  return () => { listener = undefined; channel?.close(); channel = undefined; };
}
async function sessionRequest(config: Config, path: string, method: string, body?: unknown) {
  await purgeLegacyCache();
  const response = await fetch(config.control + path, { method, credentials: 'include', redirect: 'error', cache: 'no-store',
    headers: { 'Content-Type': 'application/json', 'X-Yuki-CSRF': '1' }, body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(10000) });
  if (!response.ok) throw new Error(response.status === 401 ? '账号或密码无效，或登录已失效。' : response.status === 429 ? '请求过于频繁，请稍后重试。' : '会话服务暂不可用。');
  return response.json();
}
function checkedSession(value: BrowserSession): BrowserSession {
  if (!value.token || !['admin', 'user'].includes(value.user?.role) || !value.user.id || value.expires_in !== 300) throw new Error('会话响应无效。');
  return value;
}
export async function loginSession(config: Config, username: string, password: string) {
  const stamp = ++generation;
  const value = checkedSession(await sessionRequest(config, '/v1/session', 'POST', { username, password, client: 'web' }));
  if (stamp !== generation) throw new Error('登录状态已变化。'); update(value); return value;
}
export function renewSession(config: Config): Promise<BrowserSession> {
  if (pending) return pending;
  const stamp = generation;
  pending = (async () => {
    try {
      const run = async () => checkedSession(await sessionRequest(config, '/v1/session/refresh', 'POST', {}));
      const value = typeof navigator !== 'undefined' && navigator.locks ? await navigator.locks.request('yuki-session-refresh', run) : await run();
      if (stamp !== generation) throw new Error('登录状态已变化。'); update(value); return value;
    } catch (error) { if (stamp === generation) update(); throw error; }
    finally { pending = undefined; }
  })();
  return pending;
}
export async function logoutSession(config: Config) {
  generation++;
  await sessionRequest(config, '/v1/session', 'DELETE');
  update(); channel?.postMessage('signed_out');
}
