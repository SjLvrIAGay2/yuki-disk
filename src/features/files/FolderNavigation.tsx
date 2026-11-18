import { Button } from '../../ui';

export function FolderNavigation({ path, open }: { path: string; open(path: string): void }) {
  const parts = path.split('/').filter(Boolean);
  const folders = [{ name: '根目录', path: '' }, ...parts.map((name, index) => ({
    name, path: parts.slice(0, index + 1).join('/'),
  }))];

  return <nav className="folder-navigation" aria-label="文件夹路径">
    <ol>{folders.map((folder, index) => <li key={folder.path}>
      {index > 0 && <span aria-hidden="true">/</span>}
      <Button variant="ghost" aria-current={folder.path === path ? 'location' : undefined}
        onClick={() => open(folder.path)}>{folder.name}</Button>
    </li>)}</ol>
  </nav>;
}
