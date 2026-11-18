import { AudioPlayer } from '@yuki-ui/react/audio-player';
import { VideoPlayer } from '@yuki-ui/react/video-player';
import { ImageViewer } from '@yuki-ui/react/image-viewer';
import { DocumentViewer } from '@yuki-ui/react/document-viewer';
import { fileType, previewLimit } from './fileTypes';

// The application chooses allowed formats; rendering and media controls belong to yukiUI.
export default function PreviewContent({ content, name, onError }: { content: Blob; name: string; onError(): void }) {
  const type = fileType(name);
  if (type.preview === 'image') return <ImageViewer src={content} alt={name} onError={onError} />;
  if (type.preview === 'audio') return <AudioPlayer src={content} title={name} showMetadata={false} onError={onError} />;
  if (type.preview === 'video') return <VideoPlayer src={content} title={name} showMetadata={false} onError={onError} />;
  return <DocumentViewer src={content} filename={name} format={type.documentFormat ?? 'text'} markdownOptions={{ allowImages: false }} maxFileSize={previewLimit(type)} showHeader={false} onError={onError} />;
}
