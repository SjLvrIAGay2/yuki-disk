import type { Config } from '../../api/protocol';
import { downloadTo } from '../transfers/download';
import { previewLimit, type FileType } from './fileTypes';

export async function loadPreview(config: Config, token: string, version: string, type: FileType, signal: AbortSignal, cache: boolean): Promise<Blob> {
  const buffers: ArrayBuffer[] = [];
  await downloadTo(config, token, version, signal, {
    write: async data => { buffers.push(data); }, close: async () => {}, abort: async () => { buffers.length = 0; },
  }, () => {}, 3, cache, previewLimit(type));
  signal.throwIfAborted();
  return new Blob(buffers, { type: type.mime || 'text/plain;charset=utf-8' });
}
