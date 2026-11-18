import {randomBytes,createHash,createHmac} from 'node:crypto';

// Private server producer. No request URLs, credentials, file names or exception text.
export function createRuntimeLogs({control,key,module,send=fetch}) {
 const stream=randomBytes(24).toString('hex');let sequence=0,frozen=0,dropped=0,queue=[],pending,timer,closing,closed=false;
 let policy={version:0,upload_seconds:10,min_level:'info'},delay=10000;
 const rank={info:1,warn:2,error:3};
 function add(level,code,count=1){
  if(!key||rank[level]<rank[policy.min_level])return;
  if(!['started','stopping','request_failed','probe_failed','probe_recovered','logs_dropped'].includes(code)||!rank[level])throw new Error('Unknown log event');
  const last=queue.at(-1);if(last&&last.sequence>frozen&&last.level===level&&last.code===code&&last.count+count<=1000000){last.count+=count;return}
  if(queue.length>=1000){dropped=Math.min(1000000,dropped+count);return}
  queue.push({sequence:++sequence,level,code,occurred_at:new Date().toISOString(),count});
 }
 async function flush(){
  if(!key)return;if(pending)return pending;
  pending=(async()=>{
   while(queue.length&&Date.now()-Date.parse(queue[0].occurred_at)>6*86400000){queue.shift();dropped=Math.min(1000000,dropped+1)}
   if(dropped&&queue.length<1000){const count=dropped;dropped=0;add('warn','logs_dropped',count)}
   const entries=queue.slice(0,100).map(v=>({...v}));if(entries.length)frozen=entries.at(-1).sequence;
   const body=JSON.stringify({stream,entries,policy_version:policy.version||0}),time=String(Math.floor(Date.now()/1000)),nonce=randomBytes(24).toString('hex');
   const path='/internal/v1/logs',digest=createHash('sha256').update(body).digest('hex');
   const signature=createHmac('sha256',key).update([module,'POST',path,'',digest,time,nonce].join('\n')).digest('hex');
   const response=await send(control+path,{method:'POST',headers:{'Content-Type':'application/json','X-Yuki-Identity':module,'X-Yuki-Time':time,'X-Yuki-Nonce':nonce,'X-Yuki-Signature':signature},body,redirect:'error',signal:AbortSignal.timeout(2000)});
   if(!response.ok){await response.body?.cancel();throw new Error('Log delivery unavailable')}
   const chunks=[];let size=0;for await(const chunk of response.body){size+=chunk.length;if(size>4096)throw new Error('Invalid log response');chunks.push(chunk)}
   const reply=JSON.parse(Buffer.concat(chunks).toString());
   if(!Number.isSafeInteger(reply.ack)||reply.ack<0||reply.ack>sequence||entries.length&&reply.ack!==entries.at(-1).sequence)throw new Error('Invalid log acknowledgement');
   queue=queue.filter(v=>v.sequence>reply.ack);
   if(Number.isInteger(reply.policy?.upload_seconds)&&reply.policy.upload_seconds>=2&&reply.policy.upload_seconds<=60&&rank[reply.policy.min_level])policy=reply.policy;
  })().finally(()=>{pending=undefined});return pending;
 }
 function schedule(){if(closed||!key)return;timer=setTimeout(async()=>{try{await flush();delay=policy.upload_seconds*1000}catch{delay=Math.min(60000,delay*2)}schedule()},delay).unref()}
 add('info','started');schedule();
 return {add,flush,close(){if(closing)return closing;closing=(async()=>{closed=true;clearTimeout(timer);await pending?.catch(()=>{});add('info','stopping');const until=Date.now()+3000;do{try{await flush()}catch{return}}while(queue.length&&Date.now()<until)})();return closing}};
}
