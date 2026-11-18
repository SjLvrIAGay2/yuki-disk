import { readJSON, HTTPError } from './upstream.mjs';
const cookieName = 'yuki_status_refresh';
export function createSessions(config, stopSignal) {
  let loginActive = 0;
  function cookie(request) { const value = (request.headers.cookie || '').split(';').map(v => v.trim()).find(v => v.startsWith(cookieName + '='))?.slice(cookieName.length + 1); return value && /^[a-f0-9]{48,64}$/.test(value) ? value : ''; }
  async function exchange(request, response, route, method, body) {
    const value = cookie(request);
    if(route.endsWith('/refresh') && !value) throw new HTTPError(401,'unauthorized');
    const upstream = await fetch(config.control + route, { method, redirect: 'error', signal: AbortSignal.any([stopSignal, AbortSignal.timeout(5000)]),
      headers: { 'Content-Type':'application/json', Origin:config.publicOrigin, 'X-Yuki-CSRF':'1', ...(value ? { Cookie:cookieName+'='+value } : {}) }, body:JSON.stringify({ ...body, client:'status' }) });
    if (!upstream.ok) { await upstream.body?.cancel(); throw new HTTPError([400,401,403,429].includes(upstream.status) ? upstream.status : 503,'session_unavailable'); }
    const data = await readJSON(upstream);
    const renewed = upstream.headers.getSetCookie().find(value => value.startsWith(cookieName+'='));
    if (!renewed || !/^yuki_status_refresh=[a-f0-9]*;/.test(renewed)) throw new HTTPError(503,'session_unavailable');
    if (!/;\s*Path=[^;]+/.test(renewed) || /;\s*Domain=/i.test(renewed)) throw new HTTPError(503,'session_unavailable');
    response.setHeader('Set-Cookie',renewed.replace(/(;\s*Path=)[^;]+/, '$1' + config.publicPath + '/api/session'));
    if (method==='DELETE') return { status:'signed_out' };
    if (data.user?.role!=='admin' || typeof data.token!=='string' || data.token.length>8192 || data.expires_in!==300) throw new HTTPError(503,'session_unavailable');
    return { token:data.token, expiresAt:Date.now()+300000 };
  }
  return {
    async login(body, request, response) { if(loginActive>=4) throw new HTTPError(429,'busy'); loginActive++; try { return await exchange(request,response,'/v1/session','POST',body); } finally { loginActive--; } },
    refresh(request,response) { return exchange(request,response,'/v1/session/refresh','POST',{}); },
    logout(request,response) { return exchange(request,response,'/v1/session','DELETE',{}); },
    get(request) { const token = /^Bearer ([A-Za-z0-9_.-]{1,8192})$/.exec(request.headers.authorization || '')?.[1];
      try { const value=JSON.parse(Buffer.from(token.split('.')[0],'base64url')); if(value.v!==1 || !['status-session','session'].includes(value.purpose) || value.exp*1000<=Date.now()) throw new Error(); }
      catch { throw new HTTPError(401,'unauthorized'); }
      // This is only a syntax/scope guard. Every protected read verifies PG state at control.
      return {token}; },
    clear() {},
  };
}
