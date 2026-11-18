import type { Config, DownloadPlan } from '../../api/protocol';
import type { Sink } from './download';
import { reportConnection } from '../p2p/status';
import { downloadMountedRanges } from './mountedRanges';
import type { TransferProgress } from './metrics';
export async function downloadFile(c: Config, plan: DownloadPlan, signal: AbortSignal, sink: Sink, progress: TransferProgress, token?: string, id?: string, concurrency=3, adaptive=true) {
 const source = plan.source!; const url = new URL(source.url);
 if (!c.nodes.includes(url.origin) || url.username || url.password || url.search || url.hash || url.pathname !== '/v1/content') throw new Error('拒绝未授权的文件地址');
 if(source.p2p&&source.ranges&&source.node&&token&&id&&plan.size>0&&c.connection!=='http') {await downloadMountedRanges(c,token,id,plan,signal,sink,progress,concurrency,adaptive);return;}
 reportConnection('http');
 const response = await fetch(url, { headers: { Authorization: `Bearer ${source.token}` }, signal, redirect: 'error', credentials: 'omit' });
 if (!response.ok || !response.body || response.headers.get('ETag') !== `"${source.revision}"`) throw new Error('文件已变化或暂时无法读取，请刷新后重试。');
 const reader = response.body.getReader(); let total = 0;
 try { for (;;) { signal.throwIfAborted(); const part = await reader.read(); if (part.done) break; total += part.value.byteLength; if (total > plan.size) throw new Error('文件大小已变化'); await sink.write(part.value.slice().buffer); progress(Math.min(99, total / Math.max(1, plan.size) * 100), '正在下载',{transferred:total,total:plan.size,phase:'transfer'}); } }
 finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
 if (total !== plan.size) throw new Error('文件下载不完整，请重试。');
 signal.throwIfAborted(); progress(99,'正在保存文件',{transferred:plan.size,total:plan.size,phase:'saving'}); await sink.close(); progress(100, '下载完成');
}
