import type { ServiceDetail } from '../../api/protocol';
import { bytes, rate } from './format';

export function RoutingView({ inventory }: { inventory: ServiceDetail['inventory'] }) {
  if (!inventory?.routing_state) return null;
  const numeric = (key: string) => typeof inventory[key] === 'number' ? inventory[key] as number : undefined;
  return <><h3>自动分流</h3><dl className="resource-grid">
    <div><dt>接收新数据</dt><dd>{({ ready: '可接收', full: '容量不足', read_only: '只读', unavailable: '等待资源采样' } as Record<string, string>)[String(inventory.routing_state)]}</dd></div>
    <div><dt>可分配空间</dt><dd>{bytes(numeric('schedulable_bytes'))}</dd></div>
    <div><dt>在途空间预留</dt><dd>{bytes(numeric('reserved_bytes'))}</dd></div>
    <div><dt>接收速率</dt><dd>{rate(numeric('receive_bps'))}</dd></div>
    <div><dt>发送速率</dt><dd>{rate(numeric('send_bps'))}</dd></div>
    <div><dt>读取调度带宽</dt><dd>{rate(numeric('effective_read_bps'))}（{numeric('read_bandwidth_bps') ? '配置预算' : '自动估算'}）</dd></div>
    <div><dt>写入调度带宽</dt><dd>{rate(numeric('effective_write_bps'))}（{numeric('write_bandwidth_bps') ? '配置预算' : '自动估算'}）</dd></div>
  </dl><p className="hint">分流结合容量、带宽负载和访问热度；调度带宽不代表测速结果或限速。</p></>;
}
