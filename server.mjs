import {createRuntimeLogs} from './runtime-logs.mjs';
import {serviceTelemetry} from './telemetry.mjs';
import {validateStatusService,proxyAdminStatus} from './admin-status.mjs';
import http from 'node:http';
import https from 'node:https';
import fs from 'node:fs/promises';
import {constants} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {randomBytes, timingSafeEqual} from 'node:crypto';

export async function createWebServer(config){
 const allowed=['host','port','development','cert','key','directory','monitorKey','statusService','logKey'];
 if(config.logKey!==undefined&&(typeof config.logKey!=='string'||config.logKey.length<32||config.logKey===config.monitorKey||!config.development&&config.logKey.toLowerCase().includes('development')))throw new Error('Invalid log credential');
 validateStatusService(config.statusService,config.development);
 if(Object.keys(config).some(k=>!allowed.includes(k)))throw new Error('Unknown server config field');
 if(config.monitorKey!==undefined&&(typeof config.monitorKey!=='string'||config.monitorKey.length<32))throw new Error('Invalid monitor key');
 if(!Number.isInteger(config.port)||config.port<1||config.port>65535)throw new Error('Invalid port');
 if(config.development&&!['127.0.0.1','::1','localhost'].includes(config.host))throw new Error('Development must bind loopback');
 const root=await fs.realpath(path.resolve(config.directory));
 const publicConfig=JSON.parse(await fs.readFile(path.join(root,'config.json'),'utf8'));
 if(publicConfig.p2p_brokers!==undefined&&(!Array.isArray(publicConfig.p2p_brokers)||publicConfig.p2p_brokers.length>4))throw new Error('Invalid P2P broker list');
  const originOnly=[...publicConfig.nodes,...(publicConfig.p2p_brokers||[])];
  for(const raw of originOnly){const u=new URL(raw);if(raw!==u.origin||u.username||u.password||(!config.development&&u.protocol!=='https:'))throw new Error('Unsafe public config origin')}
  // The control plane's public address may carry a path prefix when the API is
  // published behind a reverse proxy at a subpath such as /api; node and broker
  // addresses must stay bare origins because routes are appended to them.
  {const raw=publicConfig.control;const u=new URL(raw);
   const pathAllowed=(u.pathname===''||u.pathname==='/')?raw===u.origin:(!u.pathname.endsWith('/')&&raw===u.origin+u.pathname);
   if(!pathAllowed||u.username||u.password||u.search||u.hash||(!config.development&&u.protocol!=='https:'))throw new Error('Unsafe public config origin')}
 if(Boolean(publicConfig.development)!==Boolean(config.development))throw new Error('Server/public development modes must agree');
 const types={'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.css':'text/css; charset=utf-8','.json':'application/json; charset=utf-8','.svg':'image/svg+xml','.png':'image/png','.ico':'image/x-icon'};
 const files=new Map();
 async function scan(dir){for(const entry of await fs.readdir(dir,{withFileTypes:true})){const full=path.join(dir,entry.name);if(entry.isSymbolicLink())throw new Error('Symlinks forbidden in static root');if(entry.isDirectory())await scan(full);else if(entry.isFile())files.set('/'+path.relative(root,full).split(path.sep).join('/'),full)}}
 await scan(root);
 const logs=createRuntimeLogs({control:publicConfig.control,key:config.logKey,module:'web'});
 const handler=async(req,res)=>{
  const started=Date.now();
  const styleNonce=randomBytes(24).toString('base64');
  // The trusted CDN copies this response nonce onto its injected detection
  // script. Keep it independent from the style nonce exposed to UI components.
  const scriptNonce=randomBytes(24).toString('base64');
  res.setHeader('X-Content-Type-Options','nosniff');res.setHeader('Referrer-Policy','no-referrer');res.setHeader('Cache-Control','no-store');
  // connect-src must allow the control plane's exact public address, which may
  // include a path prefix, plus every node and broker origin.
  res.setHeader('Content-Security-Policy',`default-src 'self'; script-src 'self' 'nonce-${scriptNonce}'; style-src 'self' 'nonce-${styleNonce}'; img-src 'self' data: blob:; media-src blob:; worker-src 'self'; frame-src 'none'; connect-src 'self' ${originOnly.join(' ')} ${publicConfig.control}; object-src 'none'; base-uri 'none'; frame-ancestors 'none'`);
  if(!config.development)res.setHeader('Strict-Transport-Security','max-age=31536000');
  res.on('finish',()=>{if(res.statusCode>=500)logs.add('warn','request_failed')});
  if(!['GET','HEAD'].includes(req.method)){res.writeHead(405);res.end();return}
  try{
   const url=new URL(req.url,'http://localhost');let pathname=decodeURIComponent(url.pathname);
   if(await proxyAdminStatus(req,res,url,config.statusService))return;
   if(pathname==='/statusz'){const supplied=Buffer.from(req.headers['x-yuki-monitor']||'');const expected=Buffer.from(config.monitorKey||'');if(expected.length<32||supplied.length!==expected.length||!timingSafeEqual(supplied,expected)){res.writeHead(403);res.end();return}res.setHeader('Content-Type','application/json');res.end(req.method==='HEAD'?undefined:JSON.stringify({...serviceTelemetry('web',config.development),ready:true}));return}
   if(pathname==='/healthz'||pathname==='/readyz'){res.setHeader('Content-Type','application/json');res.end('{"status":"ready"}');return}
   if(pathname.includes('\0')||pathname.includes('\\')||pathname.split('/').includes('..')){res.writeHead(400);res.end();return}
   if(pathname==='/'||['/admin','/files','/shares','/trash','/transfers','/settings'].includes(pathname)||/^\/share\/[a-f0-9]{48}$/.test(pathname))pathname='/index.html';const target=files.get(pathname);if(!target){res.writeHead(404);res.end();return}
   const handle=await fs.open(target,constants.O_RDONLY|(constants.O_NOFOLLOW||0));let bytes;
   try{bytes=await handle.readFile()}finally{await handle.close()}
   if(pathname==='/index.html')bytes=Buffer.from(bytes.toString('utf8').replace('<head>',`<head><meta name="style-nonce" content="${styleNonce}">`));
   res.setHeader('Content-Type',types[path.extname(target)]||'application/octet-stream');res.setHeader('Content-Length',bytes.length);res.end(req.method==='HEAD'?undefined:bytes);
  }catch{if(!res.headersSent)res.writeHead(400);res.end()}
 };
 const server=config.development?http.createServer(handler):https.createServer({cert:await fs.readFile(config.cert),key:await fs.readFile(config.key),minVersion:'TLSv1.2'},handler);
 server.requestTimeout=10000;server.headersTimeout=5000;server.keepAliveTimeout=5000;
 server.closeLogs=()=>logs.close();
 server.on('close',()=>void logs.close());
 return server;
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 try{
  const args=process.argv.slice(2);if(args.length!==2||args[0]!=='--config')throw new Error('Usage: node server.mjs --config server.json');
  const config=JSON.parse(await fs.readFile(args[1],'utf8'));const server=await createWebServer(config);
  server.on('error',e=>{process.stderr.write(JSON.stringify({level:'ERROR',error:e.message})+'\n');process.exitCode=1});
  server.listen(config.port,config.host,()=>process.stderr.write(JSON.stringify({level:'INFO',message:'static server ready',port:config.port})+'\n'));
  let closing=false;const stop=()=>{if(closing)return;closing=true;server.close(async()=>{await server.closeLogs();process.exit(0)});setTimeout(()=>{server.closeAllConnections();process.exit(1)},10000).unref()};process.on('SIGTERM',stop);process.on('SIGINT',stop);
 }catch(e){process.stderr.write(JSON.stringify({level:'ERROR',error:e.message})+'\n');process.exitCode=1}
}
