import { textPreviewFormat } from './textTypes';
type FileKind = 'folder' | 'image' | 'text' | 'code' | 'pdf' | 'audio' | 'video' | 'archive' | 'document' | 'spreadsheet' | 'presentation' | 'file';
type PreviewKind = 'image' | 'text' | 'pdf' | 'docx' | 'audio' | 'video';
export type FileType = { kind: FileKind; label: string; preview?: PreviewKind; mime?: string; documentFormat?: 'text' | 'code' | 'markdown' | 'pdf' | 'docx' };
const textTypes: Record<'text' | 'code' | 'markdown', FileType> = {
  text: { kind: 'text', label: '文本', preview: 'text', documentFormat: 'text' },
  code: { kind: 'code', label: '代码', preview: 'text', documentFormat: 'code' },
  markdown: { kind: 'text', label: 'Markdown', preview: 'text', documentFormat: 'markdown' },
};
const types: Record<string, FileType> = Object.create(null);
function register(extensions: string, type: FileType) { for (const extension of extensions.split(' ')) types[extension] = type; }
register('zip rar 7z tar gz bz2 xz tgz zst', { kind: 'archive', label: '压缩包' });
register('doc odt rtf pages', { kind: 'document', label: '文档' });
register('docx', { kind: 'document', label: 'Word 文档', preview: 'docx', mime: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', documentFormat: 'docx' });
register('xls xlsx ods numbers', { kind: 'spreadsheet', label: '表格' });
register('ppt pptx odp key', { kind: 'presentation', label: '演示文稿' });
register('pdf', { kind: 'pdf', label: 'PDF', preview: 'pdf', mime: 'application/pdf', documentFormat: 'pdf' });
for (const [extension, mime] of Object.entries({ png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg', gif: 'image/gif', webp: 'image/webp', avif: 'image/avif', bmp: 'image/bmp', ico: 'image/x-icon' })) {
  register(extension, { kind: 'image', label: '图片', preview: 'image', mime });
}
for (const [extension, mime] of Object.entries({ mp3: 'audio/mpeg', wav: 'audio/wav', ogg: 'audio/ogg', flac: 'audio/flac', m4a: 'audio/mp4', aac: 'audio/aac', opus: 'audio/ogg' })) {
  register(extension, { kind: 'audio', label: '音频', preview: 'audio', mime });
}
// Container recognition does not promise that every codec inside it is decodable.
register('avi wmv asf flv mpg mpeg m2ts vob rm rmvb', { kind: 'video', label: '视频' });
for (const [extension, mime] of Object.entries({ mp4: 'video/mp4', webm: 'video/webm', ogv: 'video/ogg', mov: 'video/quicktime', m4v: 'video/mp4', mkv: 'video/x-matroska', '3gp': 'video/3gpp', '3g2': 'video/3gpp2' })) {
  register(extension, { kind: 'video', label: '视频', preview: 'video', mime });
}
export function fileType(name: string, directory = false): FileType {
  if (directory) return { kind: 'folder', label: '文件夹' };
  const format = textPreviewFormat(name);
  if (format) return textTypes[format];
  return types[name.toLowerCase().split('.').pop() || ''] || { kind: 'file', label: '文件' };
}
export function previewLimit(type: FileType): number { return type.preview === 'text' ? 2 * 1024 * 1024 : 50 * 1024 * 1024; }
