import { Fragment, useId, useState } from 'react';
import { Button, Table, TableHeader, TableBody, TableRow, TableHead, TableCell, Checkbox, EmptyState, DropdownMenu, DropdownMenuItem } from '../../ui';
import type { Config, Entry } from '../../api/protocol';
import { bytes, date } from '../../lib/format';
import { FileIcon } from './FileIcon';
import { VersionList } from './VersionList';
import { versionEntry } from './FileVersionCards';

type Props = { config: Config; token: string; onChanged(): void; items: Entry[]; selected: Set<string>; toggle(path: string): void; open(path: string): void; preview(entry: Entry): void; save(entry: Entry): void; action(kind: string, paths: string[]): void; density?: 'comfortable'|'compact'; showPath?: boolean; loading?: boolean };
export function FileTable(props: Props) {
  return <div className={`file-table density-${props.density || 'comfortable'}`}><Table><TableHeader><TableRow><TableHead>选择</TableHead><TableHead>名称</TableHead><TableHead>大小</TableHead><TableHead>修改时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>
    {props.items.map(entry => <FileRow key={entry.path} {...props} entry={entry} />)}
  </TableBody></Table>{!props.loading && !props.items.length && <EmptyState title="暂无文件" description="上传文件或新建文件夹。" />}</div>;
}
function FileRow({ entry, config, token, onChanged, selected, toggle, open, preview, save, action, showPath = true }: Props & { entry: Entry }) {
  const [expanded, setExpanded] = useState(false); const panelID = useId();
  const expandable = !entry.is_dir && !entry.readonly;
  function activate() {
    if (entry.is_dir) open(entry.path);
    else if (expandable && (entry.versions > 1 || expanded)) setExpanded(value => !value);
    else preview(entry);
  }
  return <Fragment><TableRow className={selected.has(entry.path) ? 'selected' : ''}>
    <TableCell><Checkbox aria-label={`选择 ${entry.name}`} checked={selected.has(entry.path)} onCheckedChange={() => toggle(entry.path)} /></TableCell>
    <TableCell><div className="file-cell"><FileIcon name={entry.name} directory={entry.is_dir} /><div>
      <Button variant="ghost" className="entry-name" aria-expanded={expandable && (entry.versions > 1 || expanded) ? expanded : undefined} aria-controls={expandable ? panelID : undefined} onClick={activate}>{entry.name}</Button>
      <small>{showPath ? entry.path : entry.is_dir ? '文件夹' : '文件'}{entry.readonly ? ' · 只读' : !entry.is_dir && ` · ${entry.versions} 个版本`}</small></div></div></TableCell>
    <TableCell>{entry.is_dir ? '—' : bytes(entry.size)}</TableCell><TableCell>{date(entry.modified_at || entry.created_at)}</TableCell>
    <TableCell className="actions"><div className="desktop-file-actions">{!entry.is_dir && <><Button variant="ghost" onClick={() => save(entry)}>下载</Button>{expandable && <Button variant="ghost" aria-expanded={expanded} aria-controls={panelID} onClick={() => setExpanded(value => !value)}>{expanded ? '收起版本' : '版本'}</Button>}</>}
      {!entry.readonly && <><Button variant="ghost" onClick={() => action('rename', [entry.path])}>重命名</Button><Button variant="ghost" onClick={() => action('trash', [entry.path])}>删除</Button></>}{!entry.is_dir && !entry.readonly && <Button variant="ghost" onClick={() => action('visibility', [entry.path])}>分享</Button>}
    </div><div className="mobile-file-actions"><DropdownMenu trigger={<Button variant="ghost" aria-label={`${entry.name} 的操作`}>更多</Button>}>
      <DropdownMenuItem onSelect={()=>activate()}>{entry.is_dir?'打开文件夹':'查看文件'}</DropdownMenuItem>
      {!entry.is_dir && <DropdownMenuItem onSelect={()=>save(entry)}>下载</DropdownMenuItem>}
      {expandable && <DropdownMenuItem onSelect={()=>setExpanded(v=>!v)}>{expanded?'收起版本':'版本'}</DropdownMenuItem>}
      {!entry.readonly && <><DropdownMenuItem onSelect={()=>action('rename',[entry.path])}>重命名</DropdownMenuItem><DropdownMenuItem destructive onSelect={()=>action('trash',[entry.path])}>删除</DropdownMenuItem></>}
      {!entry.readonly&&!entry.is_dir&&<DropdownMenuItem onSelect={()=>action('visibility',[entry.path])}>分享</DropdownMenuItem>}
    </DropdownMenu></div></TableCell>
  </TableRow>{expanded && expandable && <TableRow className="file-versions-row"><TableCell colSpan={5}><section id={panelID} aria-label={`${entry.name} 的版本`} className="file-versions-panel">
    <VersionList config={config} token={token} path={entry.path} refresh={`${entry.version_id}:${entry.versions}`} onChanged={onChanged} preview={version => preview(versionEntry(entry, version))} save={(id, name) => save({ ...entry, version_id: id, name })} />
  </section></TableCell></TableRow>}</Fragment>;
}
