#!/usr/bin/env python3
"""Generate one deployment's distinct secrets; never overwrite existing config output."""
import argparse
import json
import os
from pathlib import Path
import secrets
from urllib.parse import urlsplit

p=argparse.ArgumentParser();p.add_argument('--topology',required=True);p.add_argument('--output',required=True);a=p.parse_args()
t=json.loads(Path(a.topology).read_text());out=Path(a.output);out.mkdir(mode=0o700,parents=True,exist_ok=False)
dev=t.get('development',False);password,session,cap,maintenance=[secrets.token_hex(32) for _ in range(4)];nodes=[];log_keys={k:secrets.token_hex(32) for k in ['web','status']}
def write(name,obj):
    f=out/name;f.parent.mkdir(parents=True,exist_ok=True);fd=os.open(f,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,'w') as handle:json.dump(obj,handle,indent=2);handle.write('\n')
for n in t['nodes']:
    if not n['id'].replace('-','').replace('_','').isalnum():raise SystemExit('Node ID must be alphanumeric with hyphen/underscore')
    key=secrets.token_hex(32);nodes.append({'id':n['id'],'url':n['url'],'tier':n['tier'],'key':key,**{k:n.get(k,0) for k in ['read_bandwidth_bps','write_bandwidth_bps','reserve_bytes']}})
    write(n['id']+'/node.json',{'monitor_key':secrets.token_hex(32),'id':n['id'],'listen':n['listen'],'data':n['data'],'control':t['control']['url'],'key':key,'capability_key':cap,'maintenance_key':maintenance,'development':dev,'sources':[x['url'] for x in t['nodes']],'origins':[t['web_origin']],'tls_cert':n.get('tls_cert',''),'tls_key':n.get('tls_key',''),'storage_mode':n.get('storage_mode','filesystem'),'capacity_bytes':n.get('capacity_bytes',0),**({'container_file':n.get('container_file','blocks.pack')} if n.get('storage_mode')=='container' else {})})
write('control.json',{'log_keys':log_keys,'bootstrap_admin':{'username':t.get('admin_username','admin'),'password':secrets.token_hex(32)},'legacy_username':t.get('initial_username','user'),'monitor_key':secrets.token_hex(32),'listen':t['control']['listen'],'database':t['database'],'development':dev,'password':password,'session_key':session,'capability_key':cap,'maintenance_key':maintenance,'nodes':nodes,'origins':[t['web_origin']],'replicas':2,'trash_seconds':604800,'gc_seconds':86400,'upload_seconds':900,'cold_seconds':604800,'tls_cert':t['control'].get('tls_cert',''),'tls_key':t['control'].get('tls_key','')})
write('maintenance.json',{'monitor_key':secrets.token_hex(32),'control':t['control']['url'],'key':maintenance,'nodes':[n['url'] for n in nodes],'development':dev,'listen':'127.0.0.1:8090'})
write('cli.json',{'username':t.get('initial_username','user'),'control':t['control']['url'],'password':password,'nodes':[n['url'] for n in nodes],'development':dev})
write('web/config.json',{'control':t['control']['url'],'nodes':[n['url'] for n in nodes],'development':dev})
write('web-server.json',{'logKey':log_keys['web'],'monitorKey':secrets.token_hex(32),'host':'127.0.0.1' if dev else '0.0.0.0','port':urlsplit(t['web_origin']).port or (80 if dev else 443),'development':dev,'directory':'/opt/yukidisk/web/dist','cert':'/etc/yukidisk/tls/fullchain.pem','key':'/etc/yukidisk/tls/privkey.pem'})
write('status-log.json',{'logKey':log_keys['status']})
print(f'Wrote private service configs to {out}; only web/config.json is public. Password is in cli.json.')
