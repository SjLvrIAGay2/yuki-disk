import { useEffect, useRef, useState, type FormEvent } from 'react';
import { Alert, Button, Input } from '../../ui';
import { APIError, request } from '../../api/client';
import type { Session } from '../../api/protocol';

export function LoginForm({ onLogin }: { onLogin(session: Session): void }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState(''); const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  const controller = useRef<AbortController | undefined>(undefined);
  useEffect(() => () => controller.current?.abort(), []);
  async function login(event: FormEvent) {
    event.preventDefault(); if (busy) return;
    const abort = new AbortController(); controller.current = abort; setBusy(true); setError('');
    try { onLogin(await request<Session>('/api/session', { method: 'POST', body: { username, password }, signal: abort.signal })); }
    catch (cause) { if (!abort.signal.aborted) setError(cause instanceof APIError && cause.status === 401 ? '访问密码不正确。' : cause instanceof APIError && cause.status === 429 ? '尝试过于频繁，请稍后重试。' : '登录暂时不可用，请稍后重试。'); }
    finally { setPassword(''); setBusy(false); }
  }
  return <><p>管理员登录后查看内部详情和维护记录。公开状态无需登录。</p>
    <form className="login-form" onSubmit={event => void login(event)}><Input label="管理员账号" autoComplete="username" value={username} onChange={event => setUsername(event.target.value)} required /><Input id="status-password" aria-label="访问密码" type="password" autoComplete="current-password" placeholder="访问密码" required value={password} onChange={event => setPassword(event.target.value)} />
      <Button type="submit" disabled={busy}>{busy ? '正在登录…' : '登录'}</Button></form>{error && <Alert variant="danger" title={error} />}</>;
}
