export function select<T>(previous: Set<T>, value: T, checked = !previous.has(value)): Set<T> {
  const next = new Set(previous);
  if (checked) next.add(value); else next.delete(value);
  return next;
}

export function outerPaths<T>(items: T[], path: (item: T) => string): T[] {
  const roots: T[] = [];
  for (const item of [...items].sort((a, b) => path(a) < path(b) ? -1 : path(a) > path(b) ? 1 : 0)) {
    if (!roots.some(root => path(item) === path(root) || path(item).startsWith(path(root) + '/'))) roots.push(item);
  }
  return roots;
}
