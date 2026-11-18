import { Dialog } from '../../ui';
import type { Config } from '../../api/protocol';
import { SharesPanel } from './SharesPanel';
export function ShareDialog({ config, token, version, onClose }: { config: Config; token: string; version: string; onClose(): void }) {
  return <Dialog open title="分享文件" onOpenChange={open => { if (!open) onClose(); }}><SharesPanel config={config} token={token} version={version} /></Dialog>;
}
