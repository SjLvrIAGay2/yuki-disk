import { CHUNK_SIZE, type Config, type Target } from './protocol';
import { accessToken, renewSession, sameSession } from '../auth/session';

// Addresses the client appends routes to must be bare origins: a path would be
// silently replaced when the route is concatenated. This covers node origins and
// P2P brokers.
export function validBase(raw: string, development: boolean): boolean {
  return validAddress(raw, development, false);
}
// The control plane's own public address is used as a request prefix rather than
// having routes appended blindly, so it may carry a path when the API is
// published behind a reverse proxy at a subpath such as /api. Credentials,
// query and fragment stay forbidden, and a trailing slash is rejected so the
// prefix concatenates predictably.
export function validControlBase(raw: string, development: boolean): boolean {
  return validAddress(raw, development, true);
}
function validAddress(raw: string, development: boolean, allowPath: boolean): boolean {
  try {
    const u = new URL(raw);
    if (!(u.protocol === 'https:' || development && u.protocol === 'http:' && ['127.0.0.1', 'localhost', '[::1]'].includes(u.hostname))) return false;
    if (u.username || u.password || u.search || u.hash) return false;
    if (u.pathname === '/' || u.pathname === '') return raw === u.origin;
    // A path prefix is only valid where the caller concatenates routes onto it.
    return allowPath && !u.pathname.endsWith('/') && raw === u.origin + u.pathname;
  } catch { return false; }
}
export function validateConfig(c: Config): Config {
	if (c.p2p_brokers !== undefined && (!Array.isArray(c.p2p_brokers) || c.p2p_brokers.length > 4 || c.p2p_brokers.some(b => !validBase(b, c.development)))) throw new Error('连接服务地址不安全');
  if (!validControlBase(c.control, c.development) || !Array.isArray(c.nodes) || !c.nodes.length || c.nodes.some(n => !validBase(n, c.development))) throw new Error('配置中的服务地址不安全');
  if (!c.development && location.protocol !== 'https:') throw new Error('生产网页必须使用 HTTPS');
  return c;
}
export class APIError extends Error {
  constructor(public status: number, message: string, public code = '') { super(message); }
}
export async function api<T>(c: Config, token: string, path: string, body?: unknown, signal?: AbortSignal, idem?: string): Promise<T> {
  const timeout = AbortSignal.timeout(30000);
  const send = (credential: string) => fetch((path.startsWith('/admin-status/') ? location.origin : c.control) + path, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json', ...(credential ? { Authorization: `Bearer ${credential}` } : {}), ...(idem ? { 'Idempotency-Key': idem } : {}) },
    body: body === undefined ? undefined : JSON.stringify(body), signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
    redirect: 'error', credentials: 'omit',
  });
  const managed = !!token && !path.startsWith('/public/') && !path.startsWith('/v1/session') && sameSession(token);
  let r = await send(managed ? accessToken() || token : token);
  if (r.status === 401 && managed && sameSession(token)) {
    await r.body?.cancel(); const session = await renewSession(c);
    if (!sameSession(token)) throw new APIError(401, '登录状态已变化，请重新操作。', 'session_changed');
    r = await send(session.token);
  }
  const data = await r.json();
  if (!r.ok) throw new APIError(r.status, data.message || data.code, data.code);
  return data as T;
}
export function checkedTarget(c: Config, t: Target, hash: string): string {
  const u = new URL(t.url);
  if (!c.nodes.includes(u.origin) || u.username || u.password || u.search || u.hash || u.pathname !== `/v1/blocks/${hash}`) throw new Error('拒绝未授权的节点地址');
  return t.url;
}
export async function checkContract(c: Config, token: string): Promise<void> {
  const contract = await api<{ version: number; chunk_size: number; hash: string }>(c, token, '/v1/contract');
  if (contract.version !== 1 || contract.chunk_size !== CHUNK_SIZE || contract.hash !== 'sha256') throw new Error('服务端协议与网页版本不兼容');
}
