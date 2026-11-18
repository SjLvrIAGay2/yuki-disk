import { validBase } from '../../api/client';
import type { Config } from '../../api/protocol';
import { hex, sha256 } from '../../lib/hash';

export type Ticket = { broker: string; identity: string; key: string; service: string; expires_at: number };
export type ICEConfig = { ice: { urls: string[]; username?: string; credential?: string }[]; session_seconds: number };
export const randomID = () => hex(crypto.getRandomValues(new Uint8Array(24)));
const utf8 = new TextEncoder();

export class Signaling {
  private key: Promise<CryptoKey>;
  constructor(readonly ticket: Ticket, config: Config) {
    if (!validBase(ticket.broker, config.development) || !config.p2p_brokers?.includes(ticket.broker) || !/^browser-[a-f0-9]{48}$/.test(ticket.identity) || !/^[a-f0-9]{96}$/.test(ticket.key) || !/^[a-zA-Z0-9_-]{1,64}$/.test(ticket.service) || !Number.isSafeInteger(ticket.expires_at) || ticket.expires_at * 1000 <= Date.now() || ticket.expires_at * 1000 > Date.now() + 125000) throw new Error('连接授权无效');
    this.key = crypto.subtle.importKey('raw', utf8.encode(ticket.key), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
  }
  async call<T>(path: string, value: unknown, signal: AbortSignal): Promise<T> {
    signal.throwIfAborted();
    const body = JSON.stringify(value); const nonce = randomID(); const time = String(Math.floor(Date.now() / 1000));
    const digest = await sha256(utf8.encode(body).buffer);
    const text = [this.ticket.identity, 'POST', path, '', digest, time, nonce].join('\n');
    const signature = hex(new Uint8Array(await crypto.subtle.sign('HMAC', await this.key, utf8.encode(text))));
    const response = await fetch(this.ticket.broker + path, { method: 'POST', body, signal, credentials: 'omit', redirect: 'error',
      headers: { 'Content-Type': 'application/json', 'X-NAT-Identity': this.ticket.identity, 'X-NAT-Time': time, 'X-NAT-Nonce': nonce, 'X-NAT-Signature': signature } });
    if (!response.ok) throw new Error('连接服务暂时不可用');
    const data = await response.text();
    if (data.length > 65536) throw new Error('连接服务响应无效');
    return JSON.parse(data) as T;
  }
  async release(): Promise<void> {
    await this.call('/v1/browser-close', {}, AbortSignal.timeout(1500)).catch(() => {});
  }
}
