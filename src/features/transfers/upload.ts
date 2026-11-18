import { api, APIError } from '../../api/client';
import { requestBlock } from './blockTransport';
import { CHUNK_SIZE, MAX_CHUNKS, type Block, type Config, type Manifest } from '../../api/protocol';
import { sha256 } from '../../lib/hash';
import { parallel } from './queue';
import { pause } from '../../lib/pause';
import type { TransferProgress } from './metrics';

export async function upload(c: Config, token: string, file: File, path: string, signal: AbortSignal,
  progress: TransferProgress, concurrency = 3, idem: string = crypto.randomUUID()): Promise<string> {
  if (file.size > CHUNK_SIZE * MAX_CHUNKS) throw new Error('文件超过 10000 个块');
  const manifest: Manifest = { path, size: file.size, chunks: [] };
  const offsets = new Map<string, number>();
  const weights = new Map<string,number>();
  progress(0,'正在准备上传',{transferred:0,total:file.size,phase:'preparing'});
  for (let off = 0; off < file.size; off += CHUNK_SIZE) {
    signal.throwIfAborted();
    const b = await file.slice(off, off + CHUNK_SIZE).arrayBuffer();
    const hash = await sha256(b); manifest.chunks.push({ hash, size: b.byteLength }); offsets.set(hash, off);
    weights.set(hash,(weights.get(hash)||0)+b.byteLength);
    progress(Math.min(20, (off + b.byteLength) / file.size * 20), '正在准备上传');
  }
  const confirmed = new Set<string>();
  let attempt = 0;
  for (;;) {
    try {
      const plan = await api<{ upload_id: string; committed: boolean; blocks: Block[] }>(c, token, '/v1/uploads', manifest, signal, idem);
      if (plan.committed) return plan.upload_id;
      let done = 0; let transferred=0;
      progress(20,'正在上传',{transferred:0,total:file.size,phase:'transfer'});
      await parallel(plan.blocks, concurrency, async block => {
        signal.throwIfAborted();
        if (!block.skip) {
          const offset = offsets.get(block.hash); if (offset === undefined) throw new Error('无效块清单');
          const data = await file.slice(offset, offset + block.size).arrayBuffer();
          await parallel(block.targets, 2, async target => {
            const r = await requestBlock(c, token, target, block.hash, signal, data);
            if (!r.ok) throw new APIError([401, 403].includes(r.status) ? 409 : r.status, '节点传输失败，正在重新申请传输权限');
          });
        }
        // Only new, known content confirmed by the current plan or successful
        // PUTs resets the failure budget. Repeated skips or empty target lists
        // cannot keep a stalled upload retrying forever.
        if (weights.has(block.hash) && (block.skip || block.targets.length > 0) && !confirmed.has(block.hash)) {
          confirmed.add(block.hash); attempt = 0;
        }
        done++; transferred+=weights.get(block.hash)||0; progress(20 + done / Math.max(1, plan.blocks.length) * 70, block.skip ? '正在保存文件' : '正在上传',{transferred,total:file.size,phase:block.skip?'saving':'transfer'});
      });
      progress(90,'正在保存文件',{transferred:file.size,total:file.size,phase:'saving'});
      const result = await api<{ version_id: string }>(c, token, `/v1/uploads/${plan.upload_id}/commit`, {}, signal);
      progress(100, '上传完成'); return result.version_id;
    } catch (e) {
      if (signal.aborted) throw e;
      // Planning could not reserve the required replicas. Repeating the entire
      // manifest immediately adds load without transferring any file bytes.
      if (e instanceof APIError && e.code === 'insufficient_capacity') throw e;
      if (e instanceof APIError && ![409, 429, 502, 503, 504].includes(e.status)) throw e;
      if (attempt === 8) throw e;
      progress(90, `正在重试（${attempt + 1}/8）`,{transferred:0,total:file.size,phase:'retry'}); await pause(Math.min(8000, 1000 * 2 ** attempt), signal, 'abort-error');
      attempt++;
    }
  }
}
