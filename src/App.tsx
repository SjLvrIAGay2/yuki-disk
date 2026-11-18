import { Alert, Button, ToastProvider } from './ui';
import { displayError } from './lib/errors';
import { useEffect, useState } from 'react';
import { validateConfig } from './api/client';
import type { Config } from './api/protocol';
import { Login } from './auth/Login';
import { Workspace } from './features/files/Workspace';
import { connectionScope } from './features/p2p/sessionScope';
import { closeConnections } from './features/p2p/pool';
import { BatchProvider } from './features/batches/BatchProvider';
import { AdminPage } from './features/admin/AdminPage';
import { SharePage } from './features/shares/SharePage';
import { renewSession, logoutSession, subscribeSession, type BrowserSession } from './auth/session';
export function App() {
  const [config, setConfig] = useState<Config>(); const [session, setSession] = useState<BrowserSession>(); const [error, setError] = useState('');
  const [restoring, setRestoring] = useState(true); const token = session?.token || '';
  const scope = connectionScope(token);
  useEffect(() => () => closeConnections(scope), [scope]);
  useEffect(() => { const controller = new AbortController();
    fetch('/config.json', { cache: 'no-store', signal: controller.signal }).then(r => r.json()).then(c => setConfig(validateConfig(c))).catch(e => { if (!controller.signal.aborted) setError(displayError(e, '页面暂时无法打开，请稍后刷新。')); });
    return () => controller.abort();
  }, []);
  useEffect(() => subscribeSession(setSession), []);
  useEffect(() => { if (!config || location.pathname.startsWith('/share/')) return; let active = true; setRestoring(true);
    void renewSession(config).catch(() => undefined).finally(() => { if (active) setRestoring(false); });
    return () => { active = false; };
  }, [config]);
  useEffect(() => { if (!config || !session) return; const timer = setTimeout(() => void renewSession(config).catch(() => undefined), 240000); return () => clearTimeout(timer); }, [config, session]);
  useEffect(() => { if (!session || location.pathname.startsWith('/share/')) return; if (session.user.role === 'admin') history.replaceState(null, '', '/admin'); else if (location.pathname === '/admin' || location.pathname === '/') history.replaceState(null, '', '/files'); }, [session?.user.id, session?.user.role]);
  async function logout() { if (!config) return; try { await logoutSession(config); } catch { setError('退出未完成，请重试。'); } }
  if (error) return <main className="login"><Alert variant="danger" title={error} /><Button onClick={() => { setError(''); void logout(); }}>重试退出</Button></main>;
  if (!config) return <main className="login">正在打开网盘…</main>;
  const share = /^\/share\/([a-f0-9]{48})$/.exec(location.pathname);
  if (share) return <SharePage config={config} id={share[1]} />;
  if (location.pathname.startsWith('/share/')) return <main className="login">分享链接无效。</main>;
  if (restoring) return <main className="login">正在恢复登录…</main>;
  if (!session) return <Login config={config} onLogin={setSession} />;
  return <ToastProvider>{session.user.role === 'admin' ? <AdminPage config={config} token={token} logout={() => void logout()} /> : <BatchProvider key={session.user.id} config={config} token={token}><Workspace config={config} token={token} logout={() => void logout()} /></BatchProvider>}</ToastProvider>;
}
