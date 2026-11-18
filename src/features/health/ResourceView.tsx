import type { ServiceDetail } from '../../api/protocol';
import { Alert, Progress } from '../../ui';
import { bytes, date, duration, percent, rate } from './format';
import { RoutingView } from './RoutingView';

export function ResourceView({ service, stale }: { service: ServiceDetail; stale: boolean }) {
  const data = !stale ? service.telemetry : undefined; const inventory = !stale ? service.inventory : undefined;
  const disk = data?.storage; const worker = data?.maintenance;
  const number = (name: string) => typeof inventory?.[name] === 'number' ? String(inventory[name]) : '未提供';
  return <div className="resource-view">
    {!data && <Alert title={stale ? '采样已过期，等待更新。' : '详细状态暂时不可用，服务可能尚未更新。'} />}
    {service.kind === 'storage' && <><h3>存储空间</h3><dl className="resource-grid">
      <div><dt>存储模式</dt><dd>{inventory?.tier === 'hot' ? '热存储' : inventory?.tier === 'cold' ? '冷存储' : '未提供'}</dd></div>
      <div><dt>存储方式</dt><dd>{disk?.mode === 'container' ? '单文件容器' : disk?.mode === 'filesystem' ? '文件系统目录' : '未提供'}</dd></div>
      <div><dt>配置空间上限</dt><dd>{disk?.configured_bytes ? bytes(disk.configured_bytes) : disk ? '使用磁盘可用空间' : '未提供'}</dd></div>
      <div><dt>已存数据</dt><dd>{bytes(disk?.stored_bytes)}</dd></div>
      <div><dt>磁盘模式</dt><dd>{disk ? disk.read_only ? '只读' : '可读写' : '未提供'}</dd></div>
      <div><dt>总容量</dt><dd>{bytes(disk?.total_bytes)}</dd></div><div><dt>可用空间</dt><dd>{bytes(disk?.available_bytes)}</dd></div>
      <div><dt>已用空间</dt><dd>{disk ? bytes(disk.total_bytes - disk.free_bytes) : '未提供'}</dd></div>
      <div><dt>已记录副本</dt><dd>{number('replicas')}</dd></div>
      <div><dt>最近心跳</dt><dd>{date(inventory?.last_seen)}</dd></div><div><dt>最近对账</dt><dd>{date(inventory?.inventory_at)}</dd></div>
    </dl>{disk && disk.total_bytes > 0 && <Progress value={(disk.total_bytes - disk.free_bytes) / disk.total_bytes * 100} label="存储空间使用率" />}<p className="hint">{disk?.mode === 'container' ? '容器容量已扣除索引区域；小文件按存储页占用空间。' : disk?.configured_bytes ? '按配置上限计量，可用空间同时受磁盘剩余容量约束。' : '容量为节点所在存储卷的容量，共用磁盘的节点不能累加。'}</p><RoutingView inventory={inventory} /></>}
    {data && <><h3>进程资源</h3><dl className="resource-grid">
      <div><dt>运行模式</dt><dd>{data.mode === 'development' ? '开发模式' : '生产模式'}</dd></div>
      <div><dt>运行时</dt><dd>{data.runtime || '未提供'}</dd></div><div><dt>运行时间</dt><dd>{duration(data.process.uptime_seconds)}</dd></div>
      <div><dt>CPU 占用（单核=100%）</dt><dd>{percent(data.process.cpu_percent)}</dd></div>
      <div><dt>进程内存</dt><dd>{bytes(data.process.rss_bytes)}</dd></div><div><dt>堆内存</dt><dd>{bytes(data.process.heap_bytes)}</dd></div>
      <div><dt>磁盘读取速率</dt><dd>{rate(data.process.read_bytes_per_second)}</dd></div><div><dt>磁盘写入速率</dt><dd>{rate(data.process.write_bytes_per_second)}</dd></div>
      {data.process.goroutines !== undefined && <div><dt>协程数</dt><dd>{data.process.goroutines}</dd></div>}
    </dl><h3>所在系统</h3><dl className="resource-grid">
      <div><dt>CPU 数量</dt><dd>{data.host.cpu_count ?? '未提供'}</dd></div><div><dt>系统 CPU 占用</dt><dd>{percent(data.host.cpu_percent)}</dd></div>
      <div><dt>系统总内存</dt><dd>{bytes(data.host.memory_total_bytes)}</dd></div><div><dt>系统可用内存</dt><dd>{bytes(data.host.memory_available_bytes)}</dd></div>
    </dl><p className="hint">速率来自相邻采样，服务重启后重新采样。系统资源指服务所在运行环境。</p></>}
    {worker && <><h3>维护工作</h3><dl className="resource-grid">
      <div><dt>工作状态</dt><dd>{{ idle: '等待任务', executing: '正在执行任务', disconnected: '控制面未连接' }[worker.state]}</dd></div>
      <div><dt>本次启动已完成</dt><dd>{worker.completed_tasks ?? '未提供'}</dd></div><div><dt>本次启动失败</dt><dd>{worker.failed_tasks ?? '未提供'}</dd></div>
      <div><dt>最近联系控制面</dt><dd>{worker.last_contact_seconds ? new Date(worker.last_contact_seconds * 1000).toLocaleString() : '尚未联系'}</dd></div>
    </dl></>}
    {service.kind === 'control' && <><h3>元数据与任务</h3><dl className="resource-grid">
      {[['blocks', '有效块数'], ['versions', '文件版本数'], ['replication_factor', '目标副本数'], ['queued_tasks', '等待任务'], ['running_tasks', '执行中任务'], ['failed_tasks', '失败任务'], ['database_connections', '数据库连接数'], ['database_in_use', '使用中连接'], ['database_limit', '连接上限']].map(([key, label]) => <div key={key}><dt>{label}</dt><dd>{number(key)}</dd></div>)}
    </dl></>}
    {data && <p className="hint">采样时间：{date(data.sampled_at)}</p>}
  </div>;
}
