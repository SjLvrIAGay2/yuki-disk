import { useEffect, useState } from 'react';
import { Alert, Button, Input, Select, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, EmptyState, Progress } from '../../ui';
import { api } from '../../api/client';
import type { Config } from '../../api/protocol';
import { displayError } from '../../lib/errors';
type Share = { id: string; version_id: string; kind: string; access_key?: string; expires_at: string | null; revoked_at: string | null };
export function SharesPanel({ config, token, version }: { config: Config; token: string; version?: string }) {
  const [items, setItems] = useState<Share[]>([]); const [kind, setKind] = useState('user'); const [hours, setHours] = useState(24); const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [key, setKey] = useState(() => crypto.randomUUID());
  const [permanent,setPermanent]=useState(false); const [defaults,setDefaults]=useState({user_share_seconds:86400,api_share_seconds:86400});
  const [loaded,setLoaded]=useState(false);
  const visible = items.filter(item => !version || item.version_id === version);
  useEffect(()=>{void run(async()=>{const v=await api<typeof defaults>(config,token,'/v1/preferences');setDefaults(v);setHours(v.user_share_seconds/3600)})},[config]);
  const change = () => setKey(crypto.randomUUID());
  async function load() { const value = await api<{ items: Share[] }>(config, token, '/v2/shares'); setItems(value.items); setLoaded(true); }
  async function run(action: () => Promise<unknown>) { setBusy(true); setError(''); try { await action(); await load(); } catch (e) { setError(displayError(e, '分享操作未完成，请重试。')); } finally { setBusy(false); } }
  useEffect(() => { void run(load); }, [config, token, version]);
  return <section><h2>文件分享</h2>{error && <Alert variant="danger" title={error} />}
    {version && <form onSubmit={e => { e.preventDefault(); void run(async () => { await api(config, token, '/v2/shares', { version_id: version, kind, expires_in: kind === 'api' && permanent ? 0 : Math.round(hours * 3600), permanent:kind==='api'&&permanent, password: kind === 'user' ? password : '' }, undefined, key); change(); setPassword(''); }); }}>
      <Select label="分享类型" value={kind} onValueChange={v => { if(!v)return;setKind(v);setHours((v==='api'?defaults.api_share_seconds:defaults.user_share_seconds)/3600||24);setPermanent(v==='api'&&defaults.api_share_seconds===0); change(); }} options={[{ value: 'user', label: '用户分享链接' }, { value: 'api', label: '授权 API 文件下载' }]} />
      <Select label="授权期限" value={permanent&&kind==='api'?'permanent':'limited'} onValueChange={v=>{if(!v)return;setPermanent(v==='permanent');change()}} options={kind==='api'?[{value:'limited',label:'自定义时长'},{value:'permanent',label:'永久（直到撤销）'}]:[{value:'limited',label:'自定义时长'}]}/>
      {!(permanent&&kind==='api')&&<Input label="有效小时数" type="number" min={1/60} step="any" max={kind==='api'?876000:720} value={hours} onChange={e => { setHours(Number(e.target.value)); change(); }} />}
      {kind === 'user' && <Input label="访问凭据（可选）" type="password" value={password} onChange={e => { setPassword(e.target.value); change(); }} />}
      <p>仅分享当前文件版本。移动不影响分享；放入回收站或永久删除后撤销，恢复文件不会恢复分享。</p><Button type="submit" disabled={busy}>创建分享</Button></form>}
    {!loaded && !error && <Progress value={null} label="正在加载分享…"/>}
    {loaded && !error && !visible.length && <EmptyState title="暂无分享文件" description={version?'此文件还没有分享，可在上方创建。':'在文件页选择文件并点击“分享”，即可创建分享链接。'}/>}
    {!!visible.length && <Table><TableHeader><TableRow><TableHead>类型</TableHead><TableHead>有效期</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>{visible.map(item => {
      const active = !item.revoked_at && (!item.expires_at || Date.parse(item.expires_at) > Date.now());
      return <TableRow key={item.id}><TableCell>{item.kind === 'user' ? '用户分享' : 'API 下载授权'}</TableCell><TableCell>{item.expires_at?new Date(item.expires_at).toLocaleString():'永久'}</TableCell><TableCell>{item.revoked_at ? '已撤销' : active ? '有效' : '已过期'}</TableCell><TableCell>
        <Button disabled={!active} onClick={() => void run(() => navigator.clipboard.writeText(item.kind === 'user' ? `${location.origin}/share/${item.id}#${item.access_key}` : item.id))}>{item.kind === 'user' ? '复制分享链接' : '复制授权编号'}</Button>
        <Button disabled={busy || !active} onClick={() => void run(() => api(config, token, `/v2/shares/${item.id}/revoke`, {}))}>撤销</Button></TableCell></TableRow>;
    })}</TableBody></Table>}<Button disabled={busy} onClick={() => void run(load)}>刷新分享</Button>
  </section>;
}
