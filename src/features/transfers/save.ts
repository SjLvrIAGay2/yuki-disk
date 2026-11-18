import type { Sink } from './download';
const folderName = 'yukidisk-download-staging-v1';
const memoryLimit = 256 * 1024 * 1024;

function publish(name: string, blob: Blob, cleanup = async () => {}) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a'); link.href = url; link.download = name;
  try { document.body.append(link);link.click(); } catch(e) { URL.revokeObjectURL(url); throw e; } finally { link.remove(); }
  setTimeout(() => { URL.revokeObjectURL(url); void cleanup().catch(()=>{}); }, 600000);
}
async function hold(name: string) {
  let release!: () => void;
  const pending = new Promise<void>(resolve => {release=resolve;});
  await new Promise<void>((resolve,reject) => {
    void navigator.locks.request(`yukidisk-download:${name}`,async()=>{resolve();await pending;}).catch(reject);
  });
  return release;
}
export async function clearDownloadStaging() {
  if(!navigator.storage?.getDirectory || !navigator.locks)return;
  const root=await navigator.storage.getDirectory();
  let folder: FileSystemDirectoryHandle;
  try { folder=await root.getDirectoryHandle(folderName); } catch(e) { if(e instanceof DOMException&&e.name==='NotFoundError')return;throw e; }
  // Only this feature's temporary files, never a different active tab's file.
  for await(const name of (folder as FileSystemDirectoryHandle & {keys():AsyncIterableIterator<string>}).keys()) {
    if(!/^[a-f0-9-]{36}\.part$/.test(name))continue;
    await navigator.locks.request(`yukidisk-download:${name}`,{ifAvailable:true},async lock=>{if(lock)await folder.removeEntry(name);});
  }
}
export async function openDestination(name: string): Promise<{ sink: Sink; maxSize: number }> {
  if(navigator.storage?.getDirectory && navigator.locks) {
    const root=await navigator.storage.getDirectory();const folder=await root.getDirectoryHandle(folderName,{create:true});
    const temporary=crypto.randomUUID()+'.part';const release=await hold(temporary);
    let writer: FileSystemWritableFileStream | undefined;let released=false;
    const cleanup=async()=>{if(released)return;try{await folder.removeEntry(temporary);}catch(e){if(!(e instanceof DOMException&&e.name==='NotFoundError'))throw e;}finally{released=true;release();}};
    try {
      const file=await folder.getFileHandle(temporary,{create:true});writer=await file.createWritable();
      const estimate=await navigator.storage.estimate();
      const maxSize=estimate.quota===undefined?Number.MAX_SAFE_INTEGER:Math.max(0,estimate.quota-(estimate.usage||0)-16*1024*1024);
      let finished=false;
      return {maxSize,sink:{
        write:async data=>{if(finished)throw new Error('下载已结束');await writer!.write(data);},
        abort:async()=>{if(!finished){finished=true;await writer!.abort().catch(()=>{});}await cleanup();},
        close:async()=>{if(finished)throw new Error('下载已结束');try{await writer!.close();finished=true;publish(name,await file.getFile(),cleanup);}catch(e){await writer!.abort().catch(()=>{});await cleanup();throw e;}},
      }};
    } catch(e) {await writer?.abort().catch(()=>{});await cleanup();throw e;}
  }
  const buffers: ArrayBuffer[] = [];let size=0;let finished=false;
  return { maxSize: memoryLimit, sink: {
    write: async b => {if(finished)throw new Error('下载已结束');if(size+b.byteLength>memoryLimit)throw new Error('浏览器下载空间不足');size+=b.byteLength;buffers.push(b);},
    abort: async () => { finished=true;buffers.length = 0; },
    close: async () => {if(finished)throw new Error('下载已结束');finished=true;const blob=new Blob(buffers);buffers.length=0;publish(name,blob);},
  } };
}
