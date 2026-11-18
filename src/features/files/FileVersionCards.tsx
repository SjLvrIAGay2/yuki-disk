import { Badge, Button, Card, Checkbox } from '../../ui';
import type { Entry, Version } from '../../api/protocol';
import { bytes, date } from '../../lib/format';
import { FileIcon } from './FileIcon';

type Props = { items: Version[]; selected: Set<string>; select(id: string, checked: boolean): void; preview(version: Version): void; save(id: string, name: string): void; remove(version: Version): void };
export function FileVersionCards({ items, selected, select, preview, save, remove }: Props) {
  return <div className="file-version-cards">{items.map((version, index) => {
    const name = version.path.split('/').pop()!;
    return <Card key={version.id} role="article" aria-label={`${name} 版本 ${items.length - index}`} className="file-version-card">
      <Checkbox aria-label={`选择版本 ${items.length - index}`} checked={selected.has(version.id)} onCheckedChange={checked => select(version.id, checked === true)} />
      <FileIcon name={name} /><div className="file-version-info">
        <Button variant="ghost" className="entry-name" onClick={() => preview(version)}>{name}</Button>
        <small>{date(version.created_at)} · {bytes(version.size)}</small>
        {index === 0 && <Badge>最新版本</Badge>}
      </div>
      <div className="file-version-actions"><Button variant="ghost" onClick={() => preview(version)}>预览</Button><Button variant="ghost" onClick={() => save(version.id, name)}>下载</Button><Button variant="ghost" disabled={!!version.trashed_at} onClick={() => remove(version)}>删除</Button></div>
    </Card>;
  })}</div>;
}
export function versionEntry(entry: Entry, version: Version): Entry {
  return { ...entry, path: version.path, name: version.path.split('/').pop()!, version_id: version.id, size: version.size, created_at: version.created_at, modified_at: version.created_at, expires_at: version.expires_at, trashed_at: version.trashed_at, trash_id: version.trash_id, versions: 1 };
}
