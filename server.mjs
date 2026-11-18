import fs from 'node:fs/promises';
import { createStatusServer } from './server/http.mjs';

try {
  if (process.argv.length !== 4 || process.argv[2] !== '--config') throw new Error('Usage: node server.mjs --config server.json');
  const application = await createStatusServer(JSON.parse(await fs.readFile(process.argv[3], 'utf8')));
  const { server, config } = application;
  let stopping = false;
  async function shutdown() {
    if (stopping) return; stopping = true;
    await application.close();
    process.stderr.write(JSON.stringify({ level: 'INFO', message: 'status server stopped' }) + '\n');
  }
  process.on('SIGINT', shutdown); process.on('SIGTERM', shutdown);
  server.on('error', error => { process.stderr.write(JSON.stringify({ level: 'ERROR', message: error.message }) + '\n'); process.exitCode = 1; void shutdown(); });
  server.listen(config.port, config.host, () => process.stderr.write(JSON.stringify({ level: 'INFO', message: 'status server ready', port: config.port }) + '\n'));
} catch (error) { process.stderr.write(JSON.stringify({ level: 'ERROR', message: error.message }) + '\n'); process.exitCode = 1; }
