import os from 'node:os';
import fs from 'node:fs';
const boot = new Date(Date.now() - process.uptime() * 1000).toISOString();
export function serviceTelemetry(service, development) {
  const memory = process.memoryUsage(); const cpu = process.cpuUsage();
  const cpus = os.cpus();
  const host = { cpu_count: cpus.length, cpu_total: cpus.reduce((total, cpu) => total + Object.values(cpu.times).reduce((a, b) => a + b, 0), 0), cpu_idle: cpus.reduce((total, cpu) => total + cpu.times.idle, 0), memory_total_bytes: os.totalmem(), memory_available_bytes: os.freemem() };
  const processInfo = { uptime_seconds: process.uptime(), cpu_seconds: (cpu.user + cpu.system) / 1e6, rss_bytes: memory.rss, heap_bytes: memory.heapUsed };
  if (process.platform === 'linux') {
    try { const available = /^MemAvailable:\s+(\d+) kB$/m.exec(fs.readFileSync('/proc/meminfo', 'utf8')); if (available) host.memory_available_bytes = Number(available[1]) * 1024; } catch { delete host.memory_available_bytes; }
    try { const io = fs.readFileSync('/proc/self/io', 'utf8'); for (const name of ['read_bytes', 'write_bytes']) { const match = new RegExp('^' + name + ':\\s+(\\d+)$', 'm').exec(io); if (match) processInfo[name] = Number(match[1]); } } catch { }
  }
  return { version: 1, service, mode: development ? 'development' : 'production', runtime: 'Node.js ' + process.versions.node, boot_id: boot, sampled_at: new Date().toISOString(), process: processInfo, host };
}
