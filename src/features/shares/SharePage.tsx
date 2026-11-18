import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Input, Progress } from '../../ui';
import { api, APIError } from '../../api/client';
import type { Config } from '../../api/protocol';
import { openDestination } from '../transfers/save';
import { downloadTo } from '../transfers/download';

type Access = { name: string; token: string; expires_at: string };
export function SharePage({ config, id }: { config: Config; id: string }) {
  const [access, setAccess] = useState<Access>(); const [password, setPassword] = useState(''); const [needsPassword, setNeedsPassword] = useState(false);
  const [error, setError] = useState(''); const [busy, setBusy] = useState(false); const [progress, setProgress] = useState<number>(); const abort = useRef<AbortController | undefined>(undefined);
  const base = `/public/v2/shares/${id}`; const secret = location.hash.slice(1);
  async function authorize(quiet = false) { if (!quiet) setBusy(true); setError(''); try { const value = await api<Access>(config, secret, `${base}/access`, { password }); setAccess(value); setNeedsPassword(false); return value; }
    catch (e) { setAccess(undefined); if (e instanceof APIError && e.code === 'share_password_required') setNeedsPassword(true);
      else setError(e instanceof APIError && e.code === 'share_revoked' ? '分享已撤销。' : e instanceof APIError && e.code === 'share_expired' ? '分享已过期。' : '分享或文件暂不可用。'); throw e; }
    finally { if (!quiet) setBusy(false); }
  }
  useEffect(() => { void authorize().catch(() => undefined); return () => abort.current?.abort(); }, [id]);
  async function download() {
    if (!access) return;
    try { const renewed = await authorize(); const target = await openDestination(renewed.name); const controller = new AbortController(); abort.current = controller; setBusy(true);
      let active = renewed.token; let renewAt = Date.now() + 240000; let pending: Promise<string> | undefined;
      const renew = () => { if (Date.now() < renewAt) return Promise.resolve(active); return pending ??= authorize(true).then(value => { active = value.token; renewAt = Date.now() + 240000; return active; }).finally(() => { pending = undefined; }); };
      await downloadTo({ ...config, connection: 'http' }, renewed.token, id, controller.signal, target.sink, setProgress, 3, false, target.maxSize, base, renew);
    } catch (e) { if (!(e instanceof DOMException && e.name === 'AbortError')) setError('下载未完成，分享可能已到期、撤销或文件暂不可用。'); }
    finally { setBusy(false); }
  }
  return <main className="login"><section className="login-card"><h1>文件分享</h1>{error && <Alert variant="danger" title={error} />}
    {needsPassword && <form onSubmit={e => { e.preventDefault(); void authorize().catch(() => undefined); }}><Input label="访问凭据" type="password" value={password} onChange={e => setPassword(e.target.value)} required /><Button type="submit" disabled={busy}>访问文件</Button></form>}
    {access && <><h2>{access.name}</h2><p>有效期至 {new Date(access.expires_at).toLocaleString()}</p><Button disabled={busy} onClick={() => void download()}>下载文件</Button></>}
    {progress !== undefined && <Progress value={progress} label={progress === 100 ? '下载完成' : '正在下载'} />}{busy && <Button onClick={() => abort.current?.abort()}>取消下载</Button>}
  </section></main>;
}
