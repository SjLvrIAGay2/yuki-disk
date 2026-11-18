import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Card } from '../../ui';
import { request } from '../../api/client';
import type { Session, Snapshot } from '../../api/protocol';
import { LoginForm } from './LoginForm';
import { EventFeed } from './EventFeed';
import { ServiceDetails } from '../health/ServiceDetails';

export function EventsPanel({ snapshot, healthError }: { snapshot?: Snapshot; healthError: boolean }) {
  const [session, setSession] = useState<Session>(); const [expired, setExpired] = useState(false);
  const [restoring,setRestoring] = useState(true); const [logoutError,setLogoutError] = useState(false);
  const generation = useRef(0); const pending = useRef<Promise<void> | undefined>(undefined); const channel = useRef<BroadcastChannel | undefined>(undefined);
  const renew = useCallback(() => {
    if(pending.current) return pending.current; const stamp=generation.current;
    pending.current=(async () => { try {
      const run=()=>request<Session>('/api/session/refresh',{method:'POST',body:{}});
      const value=navigator.locks ? await navigator.locks.request('yuki-status-refresh',run) : await run();
      if(stamp===generation.current) {setSession(value);setExpired(false);}
    } catch(error) {if(stamp===generation.current) setSession(undefined);throw error;}
    finally {pending.current=undefined;} })();return pending.current;
  },[]);
  const expire = useCallback(() => { void renew().catch(()=>setExpired(true)); }, [renew]);
  useEffect(()=>{ let active=true;void renew().catch(()=>undefined).finally(()=>{if(active)setRestoring(false);});
    channel.current=new BroadcastChannel('yuki-status-session');channel.current.onmessage=event=>{if(event.data==='signed_out'){generation.current++;setSession(undefined);setExpired(false);}};
    return()=>{active=false;generation.current++;channel.current?.close();};
  },[renew]);
  useEffect(() => {
    if (!session) return;
    const timer = setTimeout(expire, Math.max(0, session.expiresAt - Date.now()-60000)); return () => clearTimeout(timer);
  }, [session, expire]);
  async function logout() { setLogoutError(false);try {await request('/api/session',{method:'DELETE'});generation.current++;setSession(undefined);setExpired(false);channel.current?.postMessage('signed_out');}catch{setLogoutError(true);} }
  return <><Card className="session-panel"><div className="section-heading"><h2>服务详情与维护记录</h2>{session && <Button variant="ghost" onClick={()=>void logout()}>退出登录</Button>}</div>
    {logoutError && <Alert title="退出未完成，请重试。" />}
    {expired && <Alert title="会话已过期，请重新登录。" />}
    {restoring ? <p>正在恢复管理会话…</p> : !session && <LoginForm onLogin={value => { generation.current++;setSession(value); setExpired(false); }} />}
  </Card>{session && <><ServiceDetails token={session.token} onExpired={expire} snapshot={snapshot} healthError={healthError} /><Card className="events-panel"><h2>维护记录</h2><EventFeed token={session.token} onExpired={expire} /></Card></>}</>;
}
