import { fileType } from './fileTypes';
import { Icon } from '../../ui/icons';
export function FileIcon({ name, directory = false }: { name: string; directory?: boolean }) {
  const type = fileType(name, directory);
  return <span className={`file-icon file-icon-${type.kind}`} role="img" aria-label={type.label}>
    <Icon name={type.kind} />
  </span>;
}
