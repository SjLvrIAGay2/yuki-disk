import { Button, Dialog, Alert, EmptyState, Progress } from '../../ui';
import { displayError } from '../../lib/errors';
import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import type { Config, Entry } from '../../api/protocol';
import { bytes } from '../../lib/format';
import { FileIcon } from './FileIcon';
import { fileType, previewLimit } from './fileTypes';
import { connectionScope } from '../p2p/sessionScope';
import { loadPreview } from './preview';
import { Icon } from '../../ui/icons';
const PreviewContent = lazy(() => import('./PreviewContent'));

type Props = { config: Config; token: string; entry: Entry; cache: boolean; onClose(): void; save(): void };
export function PreviewDialog({ config, token, entry, cache, onClose, save }: Props) {
  const currentToken = useRef(token); currentToken.current = token; const scope = connectionScope(token);
  const type = fileType(entry.name); const supported = Boolean(type.preview); const tooLarge = entry.size > previewLimit(type);
  const [content, setContent] = useState<Blob>(); const [error, setError] = useState(''); const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (!supported || tooLarge) return;
    const controller = new AbortController();
    setContent(undefined); setError('');
    void loadPreview(config, currentToken.current, entry.version_id, type, controller.signal, cache).then(blob => {
      if (!controller.signal.aborted) setContent(blob);
    }).catch(e => { if (!controller.signal.aborted) setError(displayError(e, '暂时无法预览此文件，可重试或下载后查看。')); });
    return () => controller.abort();
  }, [config, scope, entry.version_id, cache, supported, tooLarge, type, attempt]);
  return <Dialog open title="文件预览" className="preview-dialog" onOpenChange={open => { if (!open) onClose(); }}>
    <div className="preview-summary"><FileIcon name={entry.name} /><div><strong>{entry.name}</strong><small>{type.label} · {bytes(entry.size)}</small></div><Button variant="outline" onClick={save} startIcon={<Icon name="download" size={16} />}>下载文件</Button></div>
    {!supported ? <EmptyState title="此格式暂不支持在线预览，可下载后查看。" /> : tooLarge ? <EmptyState title={`此类型的预览上限为 ${bytes(previewLimit(type))}，可下载后查看。`} /> : <div className="preview-content">
      {error ? <Alert variant="danger" title={error}><Button variant="outline" onClick={() => setAttempt(n => n + 1)}>重新加载</Button></Alert> : !content ? <Progress value={null} label="正在加载预览…" /> :
        <Suspense fallback={<Progress value={null} label="正在打开文件…" />}><PreviewContent content={content} name={entry.name} onError={() => setError(type.preview==='video'||type.preview==='audio'?'无法播放此文件，可能是浏览器不支持其编码或文件已损坏。可下载后使用本地播放器打开。':'暂时无法预览此文件，可下载后查看。')} /></Suspense>}
    </div>}
  </Dialog>;
}
