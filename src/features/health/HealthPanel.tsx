import { Alert, Badge, Button, Card, Progress, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../ui';
import type { useHealth } from './useHealth';

export function HealthPanel({ snapshot, error, refresh }: ReturnType<typeof useHealth>) {
  const stale = error || snapshot?.stale;
  const summary = error ? '状态暂时无法更新' : !snapshot || snapshot.state === 'checking' ? '正在检查服务' : snapshot.state === 'unavailable' ? '状态暂不可用' : stale ? '状态待更新' : snapshot.state === 'ready' ? '所有服务运行正常' : '部分服务不可用';
  return <section aria-label="服务健康状态">
    <Card className="overview"><div><p className="eyebrow">当前状态</p><h2>{summary}</h2>
      <p>{snapshot?.checkedAt ? `最近检查：${new Date(snapshot.checkedAt).toLocaleString()}` : '等待首次检查'}</p></div><Button variant="outline" onClick={refresh}>刷新状态</Button></Card>
    {error && <Alert variant="danger" title="无法连接状态服务，显示的数据可能已过期。" />}
    {!snapshot && !error && <Progress value={null} label="正在检查…" />}
    {snapshot && <Card className="service-table"><Table><TableHeader><TableRow><TableHead>指标</TableHead><TableHead>汇总值（约）</TableHead></TableRow></TableHeader>
      <TableBody>{[['capacity_bytes', '总容量'], ['available_bytes', '可用容量'], ['storage_used_percent', '存储使用率'], ['cpu_percent', 'CPU 使用率'], ['memory_used_percent', '内存使用率'], ['receive_bytes_per_second', '接收负载'], ['send_bytes_per_second', '发送负载']].map(([key, label]) => {
        const value = snapshot.metrics[key]; return <TableRow key={key}><TableCell>{label}</TableCell><TableCell>{stale || value === null ? '暂不可用' : key.endsWith('percent') ? `${value}%` : key.endsWith('second') ? `${(value / 1048576).toFixed(2)} MiB/s` : `${(value / 1073741824).toFixed(2)} GiB`}</TableCell></TableRow>;
      })}</TableBody></Table></Card>}
  </section>;
}
