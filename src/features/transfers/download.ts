import { downloadFile } from './downloadFile';
import { api } from '../../api/client';
import { requestBlock } from './blockTransport';
import { CHUNK_SIZE, MAX_CHUNKS, type Block, type Config, type DownloadPlan } from '../../api/protocol';
import { sha256 } from '../../lib/hash';
import { cacheBlock, cachedBlock, cacheEpoch } from './cache';
import { pause } from '../../lib/pause';
import { orderedPipeline } from './pipeline';
import type { TransferProgress } from './metrics';
import { DownloadWindow } from './downloadWindow';

export interface Sink { write(data: ArrayBuffer): Promise<void>; close(): Promise<void>; abort(): Promise<void> }
export async function downloadTo(c: Config, token: string, id: string, signal: AbortSignal, sink: Sink,
  progress: TransferProgress, concurrency = 3, useCache = true, maxSize = Number.MAX_SAFE_INTEGER, shareBase?: string, renewShare?: () => Promise<string>, adaptive = true): Promise<void> {
  try {
    const scope = cacheEpoch();
    const plan = await api<DownloadPlan>(c, token, shareBase ? `${shareBase}/download` : `/v1/download/${encodeURIComponent(id)}?manifest=true`, undefined, signal);
    if (!Number.isSafeInteger(plan.size) || plan.size < 0 || plan.size > maxSize) throw new Error('浏览器可用下载空间不足，请释放本机空间或使用 CLI');
    progress(0,'正在下载',{transferred:0,total:plan.size,phase:'transfer'});
    if (plan.source) { await downloadFile(c, plan, signal, sink, progress,token,id,concurrency,adaptive); return; }
    const window = new DownloadWindow(concurrency,adaptive,plan.size);
    if (plan.blocks.length > MAX_CHUNKS || plan.blocks.some(b => !Number.isInteger(b.size) || b.size <= 0 || b.size > CHUNK_SIZE || !/^[a-f0-9]{64}$/.test(b.hash))) throw new Error('无效块清单');
    if (plan.blocks.reduce((n, b) => n + b.size, 0) !== plan.size) throw new Error('清单大小不一致');
    let written = 0; const received = new Map<number, number>(); let lastUpdate = 0;
    const update = (index: number, bytes: number) => {
      received.set(index, Math.min(plan.blocks[index].size, bytes));
      if (performance.now() - lastUpdate < 80) return;
      lastUpdate = performance.now();
      const transferred=written+[...received.values()].reduce((a,b)=>a+b,0);
      progress(Math.min(99, transferred / plan.size * 100), '正在下载',{transferred,total:plan.size,phase:'transfer'});
    };
    let writeIndex = 0;
    await orderedPipeline(plan.blocks.length, window, signal, async (index, readSignal) => {
        const block = plan.blocks[index];
        if (useCache) {
          const hit = await cachedBlock(block.hash, scope);
          if (hit && hit.byteLength === block.size && await sha256(hit) === block.hash) { update(index, hit.byteLength); return hit; }
        }
        const data = await readBlock(c, token, id, block, readSignal, bytes => update(index, bytes), shareBase, renewShare, () => window.retry());
        if (useCache) await cacheBlock(block.hash, data, scope);
        return data;
    }, async block => {
      signal.throwIfAborted(); await sink.write(block); written += block.byteLength; received.delete(writeIndex++);window.completed(block.byteLength);
      const transferred=written+[...received.values()].reduce((a,b)=>a+b,0);
      progress(Math.min(99, transferred / plan.size * 100), '正在下载',{transferred,total:plan.size,phase:'transfer'});
    });
    signal.throwIfAborted(); progress(99,'正在保存文件',{transferred:plan.size,total:plan.size,phase:'saving'}); await sink.close(); progress(100, '下载完成');
  } catch (e) { await sink.abort().catch(() => {}); throw e; }
}
async function readBlock(c: Config, token: string, id: string, block: Block, signal: AbortSignal, received: (bytes: number) => void, shareBase?: string, renewShare?: () => Promise<string>, retry?: () => void): Promise<ArrayBuffer> {
  for (let attempt = 0; attempt < 4; attempt++) {
    signal.throwIfAborted();
    try {
      if (renewShare) token = await renewShare();
      const fresh = await api<Block>(c, token, shareBase ? `${shareBase}/blocks/${block.hash}` : `/v1/download/${encodeURIComponent(id)}/blocks/${block.hash}`, undefined, signal);
      if (fresh.hash !== block.hash || fresh.size !== block.size) throw new Error('块信息变化');
      for (const target of fresh.targets) {
        try {
          received(0);
          const r = await requestBlock(c, token, target, block.hash, signal, undefined, received);
          if (!r.ok) continue;
          if (!r.body) continue;
          const reader = r.body.getReader(); const buffer = new Uint8Array(block.size); let offset = 0;
          try {
            for (;;) {
              signal.throwIfAborted(); const part = await reader.read(); if (part.done) break;
              if (offset + part.value.length > buffer.length) throw new Error('块大小无效');
              buffer.set(part.value, offset); offset += part.value.length; received(offset);
            }
          } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
          const data = buffer.buffer;
          if (offset === block.size && await sha256(data) === block.hash) return data;
        } catch (e) { if (signal.aborted) throw e; }
      }
    } catch (e) { if (signal.aborted || attempt === 3) throw e; }
    if (attempt < 3) { retry?.();await pause(500 * 2 ** attempt, signal, 'abort-error'); }
  }
  throw new Error('所有副本均无法读取，请稍后重试');
}
