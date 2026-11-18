import { select } from '../../lib/selection';
import { ShareDialog } from '../shares/ShareDialog';
import { SharesPanel } from '../shares/SharesPanel';
import { Button, Input, Alert, Checkbox, Badge, Progress, Select, useToast } from '../../ui';
import { displayError } from '../../lib/errors';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { Config, Entry } from '../../api/protocol';
import { bytes } from '../../lib/format';
import { useEntries } from './useEntries';
import { FileTable } from './FileTable';
import { FolderNavigation } from './FolderNavigation';
import { OperationDialog } from './OperationDialog';
import { useTransfers } from '../transfers/useTransfers';
import { TransferPage } from '../transfers/TransferPage';
import { readSettings, saveSettings } from '../settings/settings';
import { SettingsPanel } from '../settings/SettingsPanel';
import { Sidebar, tabLabels as labels, type WorkspaceTab as Tab } from '../../components/Sidebar';
import { PreviewDialog } from './PreviewDialog';
import { TrashPage } from '../trash/TrashPage';
import { useBatchAction } from '../batches/BatchProvider';
import { sortEntries, sortOptions } from './sorting';
import './files.css';
import type { Settings } from '../settings/settings';

export function Workspace({ config, token, logout }: { config: Config; token: string; logout(): void }) {
  const [tab, setTab] = useState<Tab>(() => ['shares','trash','transfers','settings'].includes(location.pathname.slice(1)) ? location.pathname.slice(1) as Tab : 'files'); const [dir, setDir] = useState(''); const [search, setSearch] = useState('');
  const [selected, setSelected] = useState(new Set<string>()); const [settings, setSettings] = useState(readSettings);
  const previewConfig = useMemo(() => ({ ...config, connection: settings.connection }), [config, settings.connection]);
  const [operation, setOperation] = useState<{ kind: string; paths: string[] }>(); const { toast } = useToast();
  const [previewEntry, setPreviewEntry] = useState<Entry>();
  useEffect(() => { history.replaceState(null, '', `/${tab}`); }, [tab]);
  const pending = useBatchAction('files').task;
  const entries = useEntries(config, token, dir, false, search, tab === 'files', true);
  const sorted = useMemo(() => sortEntries(entries.items,settings.sort,settings.order),[entries.items,settings.sort,settings.order]);
  const changeSettings = (value: Settings) => { setSettings(value); saveSettings(value); };
  const transfers = useTransfers(config, token, settings, entries.refresh);
  const fileInput = useRef<HTMLInputElement>(null); const folderInput = useRef<HTMLInputElement>(null);
  useEffect(() => { folderInput.current?.setAttribute('webkitdirectory', ''); }, [tab]);
  useEffect(() => setSelected(new Set()), [dir, tab, search]);
  useEffect(() => setSelected(old => new Set([...old].filter(path => entries.items.some(item => item.path === path)))), [entries.items]);
  function open(path: string) { setDir(path); setSearch(''); }
  function toggle(path: string) { setSelected(old => select(old, path)); }
  function action(kind: string, paths = [...selected]) {
    if (pending && kind !== 'visibility') {
      setOperation({ kind: pending.kind, paths: pending.batches.flatMap(batch => (batch.body as { paths: string[] }).paths) });
    } else setOperation({ kind, paths });
  }
  async function save(id: string, name: string) { try { await transfers.save(id, name); } catch (e) { if (!(e instanceof DOMException && e.name === 'AbortError')) toast({ title: '无法开始下载', description: `${name}：${displayError(e, '请稍后重试。')}`, variant: 'danger' }); } }
  function upload(files: FileList | null) { if (files && !entries.readonly) transfers.enqueue(files, dir); }
  return <div className="shell">
    <main className={`workspace${tab === 'transfers' ? ' workspace-transfers' : ''}`}><header><Sidebar active={tab} navigate={id => { setTab(id); setSearch(''); setDir(''); }} logout={logout} /><div className="breadcrumbs"><Button variant="ghost" onClick={() => { setTab('files'); open(''); }}>网盘</Button><span>/</span>
      {tab === 'files' ? dir.split('/').filter(Boolean).map((part, i, all) => <Button variant="ghost" key={i} onClick={() => open(all.slice(0, i + 1).join('/'))}>{part} /</Button>) : labels[tab]}</div></header>
      {tab !== 'transfers' && <section className="page-heading"><div className="heading-row"><div><h1>{labels[tab]}</h1><p>{tab === 'files' ? '管理文件、文件夹及历史版本。' : tab === 'trash' ? '恢复截止前可还原文件及其版本。' : tab === 'shares' ? '查看、复制及撤销文件分享。' : '管理文件浏览与传输偏好。'}</p></div>
        {tab === 'files' && !entries.readonly && <div className="upload-actions"><Input ref={fileInput} hidden type="file" multiple onChange={e => { upload(e.target.files); e.target.value = ''; }} /><Input ref={folderInput} hidden type="file" multiple onChange={e => { upload(e.target.files); e.target.value = ''; }} />
          <Button variant="outline" onClick={() => action('mkdir', [])}>新建文件夹</Button><Button variant="outline" onClick={() => folderInput.current?.click()}>上传文件夹</Button><Button variant="primary" onClick={() => fileInput.current?.click()}>上传文件</Button></div>}
      </div></section>}
      {tab === 'files' && entries.error && <Alert variant="danger" title={entries.error}><Button variant="ghost" onClick={entries.refresh}>重试</Button></Alert>}
      {pending && !operation && <Button variant="outline" onClick={() => action(pending.kind)}>{pending.busy ? '查看操作进度' : '查看上次操作'}</Button>}
      {tab === 'files' && <section onDragOver={e => e.preventDefault()} onDrop={e => { e.preventDefault(); upload(e.dataTransfer.files); }}>
        <div className="list-toolbar"><div className="folder-location"><FolderNavigation path={dir} open={open} /><Badge>{entries.items.length} 项</Badge></div><Input aria-label="搜索文件" placeholder="搜索文件或路径" value={search} onChange={e => setSearch(e.target.value)} /><Button variant="ghost" disabled={entries.loading} onClick={entries.refresh}>刷新</Button></div>
        <div className="batch-toolbar"><Checkbox label="全选" checked={entries.items.length > 0 && entries.items.every(e => selected.has(e.path)) ? true : entries.items.some(e => selected.has(e.path)) ? "indeterminate" : false} onCheckedChange={checked => setSelected(checked === true ? new Set(entries.items.map(i => i.path)) : new Set())} />
          <span>已选择 {selected.size} 项</span><><Button variant="ghost" disabled={!selected.size || entries.items.some(e => selected.has(e.path) && e.readonly)} onClick={() => action('move')}>移动</Button><Button variant="ghost" disabled={!selected.size || entries.items.some(e => selected.has(e.path) && e.readonly)} onClick={() => action('copy')}>复制</Button><Button variant="ghost" disabled={!selected.size || entries.items.some(e => selected.has(e.path) && e.readonly)} onClick={() => action('trash')}>删除</Button></>
          <small>当前列表文件大小 {bytes(entries.items.reduce((n, e) => n + e.size, 0))}</small></div>
        <div className="file-sort"><Select label="排序依据" value={settings.sort} options={sortOptions} onValueChange={v=>changeSettings({...settings,sort:v as Settings['sort']})}/><Select label="排列顺序" value={settings.order} options={[{value:'asc',label:'升序'},{value:'desc',label:'降序'}]} onValueChange={v=>changeSettings({...settings,order:v as Settings['order']})}/></div>
        <FileTable config={config} token={token} onChanged={entries.refresh} items={sorted} selected={selected} toggle={toggle} open={open} preview={setPreviewEntry} save={(e: Entry) => void save(e.version_id, e.name)} action={action} density={settings.density} showPath={settings.showPath} loading={entries.loading} />
        {entries.loading && <Progress value={null} label="加载中…" />}{entries.cursor && <Button variant="ghost" className="load-more" disabled={entries.loading} onClick={entries.more}>加载更多</Button>}
      </section>}
      {tab === 'trash' && <TrashPage config={config} token={token} save={(id, name) => void save(id, name)} preview={setPreviewEntry} />}
      {tab === 'shares' && <SharesPanel config={config} token={token} />}
      {tab === 'settings' && <SettingsPanel value={settings} change={changeSettings} />}
      {tab === 'transfers' && <TransferPage {...transfers} />}{tab !== 'transfers' && <footer>YUKI DISK</footer>}
    </main>
    {operation?.kind === 'visibility' && <ShareDialog config={config} token={token} version={entries.items.find(e => e.path === operation.paths[0])?.version_id || ''} onClose={() => setOperation(undefined)} />}
    {operation && operation.kind !== 'visibility' && <OperationDialog {...operation} config={config} token={token} dir={dir} onClose={() => setOperation(undefined)} onDone={() => { setOperation(undefined); setSelected(new Set()); entries.refresh(); }} />}
    {previewEntry && <PreviewDialog config={previewConfig} token={token} entry={previewEntry} cache={settings.cache} onClose={() => setPreviewEntry(undefined)} save={() => void save(previewEntry.version_id, previewEntry.name)} />}
  </div>;
}
