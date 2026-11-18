import { useState } from 'react';
import { Button } from '../../ui';
import { UsersPanel } from './UsersPanel';
import { ApplicationsPanel } from './ApplicationsPanel';
import { ConfigurationPanel } from './ConfigurationPanel';
import { MountsPanel } from './MountsPanel';
import { StatusPanel } from './StatusPanel';
import type { AdminProps } from './common';
import './admin.css';
export function AdminPage({config,token,logout}:AdminProps&{logout():void}){
 const [tab,setTab]=useState('users');const props={config,token};
 return <main className="management-page"><header><h1>系统管理</h1><Button onClick={logout}>退出登录</Button></header><nav>{[['users','用户管理'],['mounts','目录挂载'],['applications','API 分享'],['status','服务详情与运行日志'],['configuration','运行配置']].map(([id,label])=><Button key={id} variant={tab===id?'primary':'ghost'} onClick={()=>setTab(id)}>{label}</Button>)}</nav>
 {tab==='users'&&<UsersPanel {...props}/>} {tab==='mounts'&&<MountsPanel {...props}/>} {tab==='applications'&&<ApplicationsPanel {...props}/>} {tab==='status'&&<StatusPanel {...props}/>} {tab==='configuration'&&<ConfigurationPanel {...props}/>}</main>;
}
