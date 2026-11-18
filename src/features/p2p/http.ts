import { CHUNK_SIZE } from '../../api/protocol';
import type { Connection } from './connection';

const utf8 = new TextEncoder();
export async function requestHTTP(connection: Connection, method: 'GET' | 'PUT', path: string, token: string, body?: ArrayBuffer, received?: (bytes: number) => void, range?: string): Promise<Response> {
  const channel = connection.channel;
  const size = body?.byteLength || 0;
  if (size > CHUNK_SIZE) throw new Error('传输内容过大');
  const metadata = utf8.encode(JSON.stringify({ method, path, headers: { Authorization: `Bearer ${token}`, ...(range ? { Range: range } : {}) }, size }));
  if (metadata.length > 8191) throw new Error('传输请求无效');
  const first = new Uint8Array(metadata.length + 1); first[0] = 4; first.set(metadata, 1); await channel.send(first);
  if (body) for (let offset = 0; offset < body.byteLength; offset += 16384) {
    const slice = new Uint8Array(body, offset, Math.min(16384, body.byteLength - offset));
    const frame = new Uint8Array(slice.length + 1); frame.set(slice, 1); await channel.send(frame);
  }
  await channel.send(new Uint8Array([1]));
  const response = await channel.read();
  if (response[0] !== 5 || response.length > 8192) throw new Error('传输响应无效');
  const header = JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(response.subarray(1))) as { status: number; size: number; headers: Record<string, string> | null };
  if (!Number.isInteger(header.status) || header.status < 200 || header.status > 599 || !Number.isSafeInteger(header.size) || header.size < 0 || header.size > CHUNK_SIZE) throw new Error('传输响应无效');
  const data = new Uint8Array(header.size); let offset = 0;
  for (;;) {
    const frame = await channel.read();
    if (frame.length === 1 && frame[0] === 1) break;
    if (frame[0] !== 0 || frame.length < 2 || offset + frame.length - 1 > data.length) throw new Error('传输内容不完整');
    data.set(frame.subarray(1), offset); offset += frame.length - 1; received?.(offset);
  }
  if (offset !== data.length) throw new Error('传输内容不完整');
  await channel.send(new Uint8Array([6]));
  const ready = await channel.read();
  if (ready.length !== 1 || ready[0] !== 8) throw new Error('连接未就绪');
  const headers = new Headers();
  for (const [name, value] of Object.entries(header.headers || {})) if (['content-type', 'etag', 'content-range'].includes(name.toLowerCase())) headers.set(name, value);
  return new Response([204, 205, 304].includes(header.status) ? null : data, { status: header.status, headers });
}
