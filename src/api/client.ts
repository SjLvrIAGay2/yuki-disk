export class APIError extends Error {
  constructor(public status: number) { super('Request failed'); }
}
export async function request<T>(path: string, options: { token?: string; method?: string; body?: unknown; signal?: AbortSignal } = {}): Promise<T> {
  const timeout = AbortSignal.timeout(10000);
  const prefix = document.querySelector<HTMLMetaElement>('meta[name="status-path"]')?.content ?? '';
  if (prefix && !/^\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*$/.test(prefix)) throw new Error('Invalid status path');
  const response = await fetch(prefix + path, {
    method: options.method ?? 'GET', credentials: path.startsWith('/api/session') ? 'same-origin' : 'omit', redirect: 'error', cache: 'no-store',
    signal: options.signal ? AbortSignal.any([timeout, options.signal]) : timeout,
    headers: { ...(path.startsWith('/api/session') ? { 'X-Yuki-CSRF':'1' } : {}), ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}), ...(options.body ? { 'Content-Type': 'application/json' } : {}) },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });
  if (!response.ok) throw new APIError(response.status);
  return response.json() as Promise<T>;
}
