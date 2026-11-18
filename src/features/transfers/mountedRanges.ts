import { api, APIError } from '../../api/client';
import { CHUNK_SIZE, type Config, type DownloadPlan, type Target } from '../../api/protocol';
import type { Sink } from './download';
import { orderedPipeline } from './pipeline';
import { requestFileRange } from './blockTransport';
import { pause } from '../../lib/pause';
import type { TransferProgress } from './metrics';
import { DownloadWindow } from './downloadWindow';
import { rangePermits } from './rangePermits';

type RangePermit = { offset: number; size: number; revision: string; range: string; target: Target };
export async function downloadMountedRanges(c: Config, token: string, id: string, plan: DownloadPlan, signal: AbortSignal, sink: Sink, progress: TransferProgress, concurrency: number, adaptive = true) {
 const window = new DownloadWindow(concurrency,adaptive,plan.size);
 const source=plan.source!;let written=0;
 async function authorize(offsets: number[], requestSignal: AbortSignal) {
  const result=await api<{ranges:RangePermit[]}>(c,token,`/v1/download/${encodeURIComponent(id)}/ranges`,{offsets,size:plan.size},requestSignal);
  if(!Array.isArray(result.ranges)||result.ranges.length!==offsets.length)throw new Error('文件授权无效');
  for(let i=0;i<offsets.length;i++){
   const p=result.ranges[i],size=Math.min(CHUNK_SIZE,plan.size-offsets[i]);
   if(p.offset!==offsets[i]||p.size!==size||p.revision!==source.revision||p.range!==`bytes=${p.offset}-${p.offset+size-1}`||p.target.node!==source.node||new URL(p.target.url).origin!==new URL(source.url).origin)throw new Error('文件授权已变化');
  }
  return result.ranges;
 }
 const permitFor = rangePermits(authorize);
 async function read(offset:number, readSignal:AbortSignal) {
  for(let attempt=0;attempt<3;attempt++) {
   readSignal.throwIfAborted();
   const permit=await permitFor(offset,readSignal);
   try{
    const response=await requestFileRange(c,token,permit.target,permit.range,readSignal);
    if(response.status===409){await response.body?.cancel().catch(()=>{});throw new APIError(409,'文件已变化，请刷新后重试。');}
    if(response.status!==206||!response.body||response.headers.get('ETag')!==`"${source.revision}"`||response.headers.get('Content-Range')!==`bytes ${permit.offset}-${permit.offset+permit.size-1}/${plan.size}`){await response.body?.cancel().catch(()=>{});throw new Error('文件范围读取失败');}
    const reader=response.body.getReader(),data=new Uint8Array(permit.size);let count=0;
    try{for(;;){readSignal.throwIfAborted();const part=await reader.read();if(part.done)break;if(count+part.value.length>data.length)throw new Error('文件大小已变化');data.set(part.value,count);count+=part.value.length;}}
    finally{await reader.cancel().catch(()=>{});reader.releaseLock();}
    if(count!==data.length)throw new Error('文件下载不完整');
    return data.buffer;
   }catch(e){if(readSignal.aborted||attempt===2||(e instanceof APIError&&e.status===409))throw e;}
   window.retry();await pause(250*(attempt+1),readSignal,'abort-error');
  }
  throw new Error('文件暂时无法读取');
 }
 await orderedPipeline(Math.ceil(plan.size/CHUNK_SIZE),window,signal,(index,s)=>read(index*CHUNK_SIZE,s),async data=>{signal.throwIfAborted();await sink.write(data);written+=data.byteLength;window.completed(data.byteLength);progress(Math.min(99,written/plan.size*100),'正在下载',{transferred:written,total:plan.size,phase:'transfer'});});
 signal.throwIfAborted();progress(99,'正在保存文件',{transferred:plan.size,total:plan.size,phase:'saving'});await sink.close();progress(100,'下载完成');
}
