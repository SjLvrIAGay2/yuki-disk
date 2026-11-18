import {createRuntimeLogs} from './runtime-logs.mjs';
import { publicSummary } from './public-summary.mjs';
import http from 'node:http';
import https from 'node:https';
import fs from 'node:fs/promises';
import { randomBytes } from 'node:crypto';
import { validateConfig } from './config.mjs';
import { createStatic } from './static.mjs';
import { createMonitor } from './monitor.mjs';
import { createSessions } from './sessions.mjs';
import { readEvents } from './events.mjs';
import { readDetails } from './details.mjs';
import { HTTPError } from './upstream.mjs';

function json(response, status, body) { response.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' }); response.end(JSON.stringify(body)); }
async function loginBody(request) {
  if (!/^application\/json(?:;|$)/i.test(request.headers['content-type'] || '')) throw new HTTPError(415, 'json_required');
  let size = 0; const chunks = [];
  for await (const chunk of request) { size += chunk.length; if (size > 4096) throw new HTTPError(413, 'body_too_large'); chunks.push(chunk); }
  let body; try { body = JSON.parse(Buffer.concat(chunks).toString('utf8')); } catch { throw new HTTPError(400, 'invalid_json'); }
  if (!body || Object.keys(body).some(key => !['username', 'password'].includes(key)) || typeof body.username !== 'string' || !body.username || typeof body.password !== 'string' || !body.password || body.password.length > 1024 || body.password.includes('\0')) throw new HTTPError(400, 'invalid_password');
  return body;
}
export async function createStatusServer(raw) {
  const config = validateConfig(raw);
  const serveStatic = await createStatic(config.directory);
  const tls = config.development ? undefined : { cert: await fs.readFile(config.cert), key: await fs.readFile(config.key), minVersion: 'TLSv1.2' };
  const logs=createRuntimeLogs({control:config.control,key:config.logKey,module:'status'});
  const stop = new AbortController(); const monitor = createMonitor(config, stop.signal,(level,code)=>logs.add(level,code));
  const sessions = createSessions(config, stop.signal);
  const attempts = new Map(); let eventRequests = 0; let detailRequests = 0;
 const publicLimits = new Map();
 function limitPublic(address) { const now = Date.now(); for (const [key, item] of publicLimits) if (item.until <= now) publicLimits.delete(key); if (!publicLimits.has(address)) { if (publicLimits.size >= 2048) throw new HTTPError(429, 'busy'); publicLimits.set(address, { count: 0, until: now + 60000 }); } if (++publicLimits.get(address).count > 120) throw new HTTPError(429, 'rate_limited'); }
  function limitLogin(address) {
    for (const [key, value] of attempts) if (value.until <= Date.now()) attempts.delete(key);
    if (!attempts.has(address)) { if (attempts.size >= 1024) throw new HTTPError(429, 'busy'); attempts.set(address, { count: 0, until: Date.now() + 60000 }); }
    if (++attempts.get(address).count > 10) throw new HTTPError(429, 'too_many_attempts');
  }
  const handler = async (request, response) => {
    const start = Date.now(); const nonce = randomBytes(24).toString('base64');
    response.setHeader('Cache-Control', 'no-store'); response.setHeader('X-Content-Type-Options', 'nosniff'); response.setHeader('Referrer-Policy', 'no-referrer');
    response.setHeader('Content-Security-Policy', `default-src 'self'; script-src 'self'; style-src 'self' 'nonce-${nonce}'; img-src 'self' data:; connect-src 'self'; frame-src 'none'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'`);
    if (!config.development) response.setHeader('Strict-Transport-Security', 'max-age=31536000');
    response.on('finish',()=>{if(response.statusCode>=500)logs.add('warn','request_failed')});
    try {
      if (stop.signal.aborted) throw new HTTPError(503, 'closing');
      if (request.headers.origin && request.headers.origin !== config.publicOrigin) throw new HTTPError(403, 'origin_denied');
      if (!request.url.startsWith('/')) throw new HTTPError(400, 'invalid_path');
      const decoded = decodeURIComponent(request.url.split('?')[0]);
      if (/[\x00-\x1f\\]/.test(decoded) || decoded.split('/').includes('..')) throw new HTTPError(400, 'invalid_path');
      const url = new URL(request.url, config.publicOrigin);
      if (request.method === 'GET' && ['/healthz', '/readyz'].includes(url.pathname)) return json(response, 200, { status: 'ready' });
      if (config.publicPath) {
        if (url.pathname === config.publicPath && ['GET', 'HEAD'].includes(request.method)) {
          response.writeHead(308, { Location: config.publicPath + '/' + url.search }); response.end(); return;
        }
        if (!url.pathname.startsWith(config.publicPath + '/')) throw new HTTPError(404, 'not_found');
        url.pathname = url.pathname.slice(config.publicPath.length);
      }
      if (url.pathname.startsWith('/api/session') && (!request.headers.origin || request.headers['x-yuki-csrf'] !== '1')) throw new HTTPError(403,'csrf_required');
      if (request.method === 'GET' && ['/healthz', '/readyz'].includes(url.pathname)) return json(response, 200, { status: 'ready' });
      if (request.method === 'GET' && url.pathname === '/api/status') { if (url.search) throw new HTTPError(400, 'invalid_query'); limitPublic(request.socket.remoteAddress); return json(response, 200, publicSummary(monitor.snapshot(), monitor.details(), config)); }
      if (request.method === 'POST' && url.pathname === '/api/session') {
        limitLogin(request.socket.remoteAddress); return json(response, 200, await sessions.login(await loginBody(request),request,response));
      }
      if (request.method === 'POST' && url.pathname === '/api/session/refresh') return json(response,200,await sessions.refresh(request,response));
      if (request.method === 'DELETE' && url.pathname === '/api/session') return json(response,200,await sessions.logout(request,response));
      if (request.method === 'GET' && url.pathname === '/api/details') {
        const session = sessions.get(request);
        if (url.search) throw new HTTPError(400, 'invalid_query');
        if (detailRequests >= 4) throw new HTTPError(429, 'busy');
        detailRequests++;
        try { return json(response, 200, await readDetails(config, monitor, session, stop.signal)); }
        finally { detailRequests--; }
      }
      if (request.method === 'GET' && url.pathname === '/api/events') {
        const session = sessions.get(request);
        if (eventRequests >= 16) throw new HTTPError(429, 'busy');
        eventRequests++;
        try { return json(response, 200, await readEvents(config, url, session, stop.signal)); }
        finally { eventRequests--; }
      }
      if (!['GET', 'HEAD'].includes(request.method)) throw new HTTPError(405, 'method_not_allowed');
      if (await serveStatic(url.pathname, request, response, nonce, config.publicPath)) return;
      throw new HTTPError(404, 'not_found');
    } catch (error) {
      if (response.destroyed) return;
      if (response.headersSent) { response.end(); return; }
      json(response, error instanceof HTTPError ? error.status : error instanceof URIError ? 400 : 503,
        { code: error instanceof HTTPError ? error.code : 'unavailable' });
    }
  };
  const server = tls ? https.createServer(tls, handler) : http.createServer(handler);
  server.requestTimeout = 10000; server.headersTimeout = 5000; server.keepAliveTimeout = 3000;
  let closing;
  return { server, config, close() {
    if (closing) return closing;
    closing = (async () => {
      stop.abort(); sessions.clear(); attempts.clear();
      const deadline = setTimeout(() => server.closeAllConnections(), 5000).unref();
      await Promise.all([monitor.close(), new Promise(resolve => server.close(resolve))]); clearTimeout(deadline); await logs.close();
    })(); return closing;
  } };
}
