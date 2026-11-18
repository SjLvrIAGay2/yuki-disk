import { useEffect, useState } from 'react';
import { APIError, request } from '../../api/client';
import type { Details, Snapshot } from '../../api/protocol';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger, Alert, Card, Progress } from '../../ui';
import { ResourceView } from './ResourceView';

export function ServiceDetails({ token, onExpired, snapshot, healthError }: { token: string; onExpired(): void; snapshot?: Snapshot; healthError: boolean }) {
  const [details, setDetails] = useState<Details>(); const [error, setError] = useState(false);
  const checkedAt = snapshot?.checkedAt;
  const healthStale = healthError || !checkedAt || !!snapshot?.stale;
  useEffect(() => {
    if (healthStale) return;
    const controller = new AbortController();
    async function load() {
      try {
        const result = await request<Details>('/api/details', { token, signal: controller.signal });
        if (!controller.signal.aborted) { setDetails(result); setError(false); }
      } catch (cause) {
        if (controller.signal.aborted) return;
        if (cause instanceof APIError && cause.status === 401) { onExpired(); return; }
        setError(true);
      }
    }
    void load(); return () => controller.abort();
  }, [token, onExpired, checkedAt, healthStale]);
  return <Card className="service-details"><h2>服务详情</h2>
    {error && <Alert variant="danger" title="服务详情暂时无法更新，将自动重试。" />}
    {!details && !error && <Progress value={null} label="正在读取服务详情…" />}
    {details?.inventoryState === 'unavailable' && <Alert title="节点模式与任务统计暂时无法读取。" />}
    {details && <Accordion type="multiple">{details.services.map(service => <AccordionItem key={service.id} value={service.id}>
      <AccordionTrigger>{service.name}</AccordionTrigger><AccordionContent><ResourceView service={service} stale={error || details.stale || healthStale || !!checkedAt && (!details.checkedAt || details.checkedAt < checkedAt)} /></AccordionContent>
    </AccordionItem>)}</Accordion>}
  </Card>;
}
