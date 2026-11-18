import { Fragment, useId, useState } from 'react';
import { VersionList } from '../files/VersionList';
import { versionEntry } from '../files/FileVersionCards';
import { Button, Checkbox, EmptyState, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../ui';
import type { Config, Entry } from '../../api/protocol';
import { bytes, date } from '../../lib/format';
import { FileIcon } from '../files/FileIcon';

type Props = { config: Config; token: string; onChanged(): void; save(id: string, name: string): void; items: Entry[]; selected: Set<string>; toggle(path: string): void; action(kind: 'restore' | 'purge', entries: Entry[]): void; preview(entry: Entry): void; };
export function TrashTable(props: Props) {
 const { items } = props;
  return <div className="file-table"><Table><TableHeader><TableRow><TableHead>选择</TableHead><TableHead>名称与原位置</TableHead><TableHead>大小</TableHead><TableHead>删除时间</TableHead><TableHead>恢复截止</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>
    {items.map(entry => <TrashRow key={entry.trash_id ?? entry.path} {...props} entry={entry} />)}
  </TableBody></Table>{!items.length && <EmptyState title="回收站为空" description="已删除的文件和文件夹将在此显示。" />}</div>;
}

function TrashRow({ entry, config, token, onChanged, save, selected, toggle, action, preview }: Props & { entry: Entry }) {
 const [expanded, setExpanded] = useState(false); const panelID = useId();
  return <Fragment><TableRow className={selected.has(entry.path) ? 'selected' : ''}>
      <TableCell><Checkbox aria-label={`选择 ${entry.name}`} checked={selected.has(entry.path)} onCheckedChange={() => toggle(entry.path)} /></TableCell>
      <TableCell><div className="file-cell"><FileIcon name={entry.name} directory={entry.is_dir} /><div>{entry.is_dir ? <strong>{entry.name}</strong> : <Button variant="ghost" className="entry-name" aria-expanded={entry.versions > 1 || expanded ? expanded : undefined} aria-controls={panelID} onClick={() => entry.versions > 1 || expanded ? setExpanded(value => !value) : preview(entry)}>{entry.name}</Button>}<small>{entry.path}{!entry.is_dir && ` · ${entry.versions} 个版本`}</small></div></div></TableCell>
      <TableCell>{entry.is_dir ? '—' : bytes(entry.size)}</TableCell><TableCell>{date(entry.trashed_at)}</TableCell><TableCell>{date(entry.expires_at)}</TableCell>
      <TableCell className="actions">{!entry.is_dir && <Button variant="ghost" aria-expanded={expanded} aria-controls={panelID} onClick={() => setExpanded(value => !value)}>{expanded ? '收起版本' : '版本'}</Button>}<Button variant="ghost" onClick={() => action('restore', [entry])}>恢复</Button><Button variant="ghost" onClick={() => action('purge', [entry])}>永久删除</Button></TableCell>
    </TableRow>{expanded && !entry.is_dir && <TableRow className="file-versions-row"><TableCell colSpan={6}><section id={panelID} aria-label={`${entry.name} 的版本`} className="file-versions-panel">
      <VersionList config={config} token={token} path={entry.path} refresh={`${entry.version_id}:${entry.versions}`} onChanged={onChanged} preview={version => preview(versionEntry(entry, version))} save={save} />
    </section></TableCell></TableRow>}</Fragment>;
}