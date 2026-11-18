import { Channel } from './channel';
import { Signaling, randomID, type ICEConfig } from './signaling';
import { pause } from '../../lib/pause';

type Route = 'direct' | 'relay';
export type Connection = { channel: Channel; route: Route; expires: number; healthy(): boolean; close(): void; maxRequests: number; keepAlive(): Promise<void> };

async function phase(sig: Signaling, mode: Route, signal: AbortSignal, lifetime: AbortSignal): Promise<Connection> {
  const settings = await sig.call<ICEConfig>('/v1/ice', { mode }, signal);
  if (!Array.isArray(settings.ice) || settings.ice.length > 4) throw new Error('连接配置无效');
  const pc = new RTCPeerConnection({ iceServers: settings.ice, iceTransportPolicy: mode === 'relay' ? 'relay' : 'all' });
  const dc = pc.createDataChannel('yukinat.http.v3', { ordered: true }); const channel = new Channel(dc, lifetime);
  const candidates: string[] = [];
  pc.onicecandidate = event => { if (event.candidate && candidates.length < 64) candidates.push(event.candidate.candidate); };
  const stop = () => { channel.close(); pc.close(); };
  signal.addEventListener('abort', stop, { once: true });
  let success = false;
  try {
    signal.throwIfAborted();
    await pc.setLocalDescription(await pc.createOffer());
    const id = randomID();
    const sdp = pc.localDescription?.sdp;
    if (!sdp || sdp.length > 16384) throw new Error('连接协商失败');
    await sig.call('/v1/offer', { id, service: sig.ticket.service, mode, protocol: 'http3', sdp }, signal);
    let after = 0; const remote: string[] = [];
    while (dc.readyState !== 'open') {
      const batch = candidates.slice(0, 16);
      const exchange = await sig.call<{ candidates: string[]; next: number }>('/v1/candidates', { id, after, candidates: batch }, signal);
      candidates.splice(0, batch.length); after = exchange.next; remote.push(...exchange.candidates);
      if (!pc.remoteDescription) {
        const result = await sig.call<{ state: string; sdp?: string }>('/v1/result', { id }, signal);
        if (result.state === 'answered' && result.sdp) await pc.setRemoteDescription({ type: 'answer', sdp: result.sdp });
      }
      if (pc.remoteDescription) for (const candidate of remote.splice(0)) await pc.addIceCandidate({ candidate, sdpMid: '0' });
      if (dc.readyState === 'closed' || pc.connectionState === 'failed') throw new Error('连接失败');
      await pause(60, signal);
    }
    const report = await pc.getStats();
    let pair: RTCStats | undefined;
    report.forEach(stat => { if (stat.type === 'transport' && stat.selectedCandidatePairId) pair = report.get(stat.selectedCandidatePairId); });
    if (!pair) report.forEach(stat => { if (stat.type === 'candidate-pair' && stat.nominated && stat.state === 'succeeded') pair = stat; });
    const selected = pair as { localCandidateId?: string; remoteCandidateId?: string } | undefined;
    if (!selected?.localCandidateId || !selected.remoteCandidateId) throw new Error('连接路径不可用');
    const route: Route = [report.get(selected.localCandidateId), report.get(selected.remoteCandidateId)].some(c => c?.candidateType === 'relay') ? 'relay' : 'direct';
    if (route !== mode) throw new Error('连接路径不匹配');
    const hello = await channel.read();
    if (hello[0] !== 7 || hello.length > 1024) throw new Error('连接版本不匹配');
    const limits = JSON.parse(new TextDecoder().decode(hello.subarray(1))) as { lifetime_seconds: number; idle_seconds: number; max_requests: number };
    if (!Number.isInteger(limits.lifetime_seconds) || limits.lifetime_seconds < 10 || limits.lifetime_seconds > 86400 || limits.idle_seconds !== 300 || limits.max_requests !== 1000000) throw new Error('连接期限无效');
    signal.throwIfAborted(); success = true;
    lifetime.addEventListener('abort', stop, { once: true });
    return { channel, route, maxRequests: limits.max_requests - 1, keepAlive: async () => {
      await channel.send(new Uint8Array([9])); const pong = await channel.read();
      if (pong.length !== 1 || pong[0] !== 10) throw new Error('连接保活失败');
    }, expires: Date.now() + (limits.lifetime_seconds - 5) * 1000, healthy: () => dc.readyState === 'open' && pc.connectionState === 'connected', close: () => { lifetime.removeEventListener('abort', stop); stop(); } };
  } finally { signal.removeEventListener('abort', stop); if (!success) stop(); }
}

export async function connect(sig: Signaling, signal: AbortSignal, lifetime: AbortSignal): Promise<Connection> {
  try { return await phase(sig, 'direct', AbortSignal.any([signal, AbortSignal.timeout(10000)]), lifetime); }
  catch (error) { if (signal.aborted) throw error; }
  return phase(sig, 'relay', AbortSignal.any([signal, AbortSignal.timeout(25000)]), lifetime);
}
