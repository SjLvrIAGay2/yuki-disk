// Fixed read-only metadata proxy. PG authorization is checked by status/control on every read.
//
// statusService is a fixed, validated base address including its optional mount
// path. It never comes from browser query parameters or forwarding headers.
// The status service assembles details by reading its own monitor cache plus the
// control plane's inventory, which can stall briefly on a small host while
// PostgreSQL connections are busy. This endpoint is only requested when an
// administrator opens the system page, so a generous ceiling is preferable to
// showing "status unavailable" for a healthy deployment.
const statusTimeoutMs = 20000;
export function validateStatusService(value, development) {
  if (value === undefined) return;
  const url = new URL(value);
  const prefix = url.pathname === '/' ? '' : url.pathname;
  if(value!==url.origin+prefix || prefix && (!/^\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*$/.test(prefix) || prefix.length>200) || url.username || url.password || url.search || url.hash || !(url.protocol==='https:' || development && url.protocol==='http:' && ['localhost','127.0.0.1','[::1]'].includes(url.hostname))) throw new Error('Invalid status service address');
}
export async function proxyAdminStatus(req,res,url,service) {
  if(!url.pathname.startsWith('/admin-status/')) return false;
  const route={'/admin-status/details':'/api/details','/admin-status/events':'/api/events'}[url.pathname];
  res.setHeader('Content-Type','application/json');
  function end(status,code){res.writeHead(status);res.end(JSON.stringify({code}));return true}
  if(!route || req.method!=='GET')return end(404,'not_found');
  if(!service)return end(503,'status_not_configured');
  if(!/^Bearer [A-Za-z0-9_.-]{1,8192}$/.test(req.headers.authorization||''))return end(401,'unauthorized');
  if(route==='/api/details' && url.search || [...url.searchParams.keys()].some(key=>key!=='after'))return end(400,'invalid_query');
  try{
    const upstream=await fetch(service+route+url.search,{headers:{Authorization:req.headers.authorization},redirect:'error',signal:AbortSignal.timeout(statusTimeoutMs)});
    const chunks=[];let size=0;for await(const chunk of upstream.body){size+=chunk.length;if(size>2*1024*1024)throw new Error('response too large');chunks.push(chunk)}
    const body=JSON.parse(Buffer.concat(chunks));res.writeHead([200,400,401,403,429,503].includes(upstream.status)?upstream.status:503);res.end(JSON.stringify(body));return true;
  }catch{return end(503,'status_unavailable')}
}
