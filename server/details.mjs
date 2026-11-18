import { controlJSON, HTTPError } from './upstream.mjs';
import { projectInventory } from './resource-details.mjs';

export async function readDetails(config, monitor, session, stopSignal) {
  let inventory; let inventoryState = 'ready';
  try {
    inventory = projectInventory(await controlJSON(config, '/v1/status', { headers: { Authorization: `Bearer ${session.token}` } }, stopSignal), config.services);
  } catch (error) {
    // Never disclose cached private telemetry without a current PG authorization.
    throw error;
  }
  const details = monitor.details();
  return { ...details, stale: monitor.snapshot().stale, inventoryState, services: details.services.map(service => ({ ...service, inventory: service.kind === 'control' ? inventory?.control : service.kind === 'storage' ? inventory?.nodes[service.id] : undefined })) };
}
