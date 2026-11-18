import { api, checkedTarget } from '../../api/client';
import type { Config, Target } from '../../api/protocol';
import { Signaling, type Ticket } from '../p2p/signaling';
import { connect } from '../p2p/connection';
import { connectionScope } from '../p2p/sessionScope';
import { acquire } from '../p2p/pool';
import { requestHTTP } from '../p2p/http';
import { reportConnection } from '../p2p/status';

export async function requestBlock(c: Config, session: string, target: Target, hash: string, signal: AbortSignal, body?: ArrayBuffer, received?: (bytes: number) => void): Promise<Response> {
  const url = checkedTarget(c, target, hash); const method = body ? 'PUT' : 'GET';
  return requestNode(c,session,target,url,method,signal,body,received);
}

export async function requestFileRange(c: Config, session: string, target: Target, range: string, signal: AbortSignal): Promise<Response> {
  const url = new URL(target.url);
  if (!c.nodes.includes(url.origin) || url.username || url.password || url.search || url.hash || url.pathname !== '/v1/content/range' || !/^bytes=\d+-\d+$/.test(range)) throw new Error('拒绝未授权的文件地址');
  return requestNode(c,session,target,url.toString(),'GET',signal,undefined,undefined,range);
}

async function requestNode(c: Config, session: string, target: Target, url: string, method: 'GET'|'PUT', signal: AbortSignal, body?: ArrayBuffer, received?: (bytes: number) => void, range?: string): Promise<Response> {
  if (target.p2p && c.connection !== 'http' && typeof RTCPeerConnection !== 'undefined' && globalThis.isSecureContext) {
    const timeout = AbortSignal.any([signal, AbortSignal.timeout(90000)]);
    let lease: Awaited<ReturnType<typeof acquire>> | undefined;
    let clean = false;
    try {
      try {
        const scope = connectionScope(session);
        const key = JSON.stringify([c.control, c.p2p_brokers, target.node, scope]);
        lease = await acquire(key, scope, timeout, async lifetime => {
          const signal = AbortSignal.any([timeout, lifetime]);
          const ticket = await api<Ticket>(c, session, `/v1/p2p/${encodeURIComponent(target.node)}`, {}, signal);
          const sig = new Signaling(ticket, c);
          try { return await connect(sig, signal, lifetime); } finally { void sig.release(); }
        }, method==='GET'?6:3);
      } catch (error) { if (signal.aborted) throw error; }
      if (lease) {
        reportConnection(lease.connection.route);
        // After application bytes are sent, only the existing upload/download retry
        // loop may retry with fresh node capabilities and the same upload identity.
        const response = await requestHTTP(lease.connection, method, new URL(url).pathname, target.token, body, received,range);
        clean = true; return response;
      }
    } finally { lease?.release(clean); }
  }
  signal.throwIfAborted(); reportConnection('http');
  return fetch(url, { method, headers: { Authorization: `Bearer ${target.token}`, ...(range ? { Range: range } : {}) }, body,
    signal: AbortSignal.any([signal, AbortSignal.timeout(30000)]), redirect: 'error', credentials: 'omit' });
}
