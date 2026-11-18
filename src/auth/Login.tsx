import { Button, Input, Alert } from '../ui';
import { displayError } from '../lib/errors';
import { useState, type FormEvent } from 'react';
import { loginSession, type BrowserSession } from './session';
import type { Config } from '../api/protocol';
export function Login({ config, onLogin }: { config: Config; onLogin(session: BrowserSession): void }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState(''); const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  async function submit(e: FormEvent) {
    e.preventDefault(); setBusy(true); setError('');
    try { const session = await loginSession(config, username, password); setPassword(''); onLogin(session); }
    catch (e) { setError(displayError(e, '登录失败，请重试。')); } finally { setBusy(false); }
  }
  return <main className="login"><div className="wordmark"><span className="logo">雪</span> yuki<span>disk</span></div>
    <section className="login-card"><h1>登录网盘</h1><form onSubmit={submit}><label htmlFor="username">账号</label>
      <Input id="username" autoComplete="username" value={username} onChange={e => setUsername(e.target.value)} required autoFocus /><label htmlFor="password">密码</label>
      <Input id="password" type="password" autoComplete="current-password" value={password} onChange={e => setPassword(e.target.value)} required />
      <Button variant="primary" type="submit" disabled={busy}>{busy ? '正在登录…' : '登录'}</Button></form>
      {error && <Alert variant="danger" title={error} />}<small>在会话有效期内可恢复登录。</small></section>
    <div className="login-art" aria-hidden="true"><div className="orbit a" /><div className="orbit b" /><div className="art-center"><strong>Yuki Disk</strong></div></div>
  </main>;
}
