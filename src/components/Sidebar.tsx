import { Button, Dialog } from '../ui';
import { useEffect, useState } from 'react';
import { Icon } from '../ui/icons';
export type WorkspaceTab = 'files' | 'transfers' | 'trash' | 'settings' | 'shares';
export const tabLabels = { files: '全部文件', transfers: '传输', trash: '回收站', shares: '文件分享', settings: '设置' };
type Props = { active: WorkspaceTab; navigate(tab: WorkspaceTab): void; logout(): void };
export function Sidebar({ active, navigate, logout }: Props) {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const media = matchMedia('(max-width: 720px)');
    const resize = () => setOpen(false);
    media.addEventListener('change', resize);
    return () => media.removeEventListener('change', resize);
  }, []);
  const content = <>
    <div className="wordmark"><span className="logo">雪</span> yuki<span>disk</span></div><div className="space-label">文件管理</div>
    <nav aria-label="主导航">{(Object.keys(tabLabels) as WorkspaceTab[]).map(id => <Button variant="ghost" key={id} className={active === id ? 'active' : ''} aria-current={active === id ? 'page' : undefined} onClick={() => { setOpen(false); navigate(id); }}>{tabLabels[id]}</Button>)}</nav>
    <div className="sidebar-note">文件存储</div><Button variant="ghost" className="logout" onClick={logout}>退出登录</Button>
  </>;
  return <>
    <aside className="desktop-sidebar">{content}</aside>
    <Button variant="outline" className="mobile-menu" aria-label="打开导航菜单" aria-expanded={open} aria-haspopup="dialog" onClick={() => setOpen(true)}>
      <Icon name="menu" size={20} /><span>菜单</span>
    </Button>
    {open && <Dialog open title="导航" className="sidebar-drawer" onOpenChange={setOpen}>{content}</Dialog>}
  </>;
}
