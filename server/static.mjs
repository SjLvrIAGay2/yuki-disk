import fs from 'node:fs/promises';
import { constants } from 'node:fs';
import path from 'node:path';

export async function createStatic(directory) {
  const root = await fs.realpath(directory); const files = new Map();
  async function scan(folder) {
    for (const entry of await fs.readdir(folder, { withFileTypes: true })) {
      const full = path.join(folder, entry.name);
      if (entry.isSymbolicLink()) throw new Error('Static symlinks forbidden');
      if (entry.isDirectory()) await scan(full);
      else if (entry.isFile()) files.set('/' + path.relative(root, full).split(path.sep).join('/'), full);
    }
  }
  await scan(root); if (!files.has('/index.html')) throw new Error('Build status before starting the server');
  const types = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon' };
  return async (pathname, request, response, nonce, publicPath = '') => {
    if (pathname === '/') pathname = '/index.html';
    const target = files.get(pathname);
    if (!target || pathname !== '/index.html' && !pathname.startsWith('/assets/')) return false;
    const handle = await fs.open(target, constants.O_RDONLY | (constants.O_NOFOLLOW || 0)); let content;
    try { content = await handle.readFile(); } finally { await handle.close(); }
    if (pathname === '/index.html') content = Buffer.from(content.toString('utf8').replace('<head>', `<head><meta name="style-nonce" content="${nonce}"><meta name="status-path" content="${publicPath}">`));
    response.writeHead(200, { 'Content-Type': types[path.extname(target)] || 'application/octet-stream', 'Content-Length': content.length });
    response.end(request.method === 'HEAD' ? undefined : content); return true;
  };
}
