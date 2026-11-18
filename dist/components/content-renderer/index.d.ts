import { CodeBlockProps } from '../code-block';
import { MarkdownProps } from '../markdown';
import { DocumentViewerProps } from '../document-viewer';
import { VideoPlayerProps } from '../video-player';
import { AudioPlayerProps } from '../audio-player';
export type ContentRendererProps = ({
    type: 'code';
} & CodeBlockProps) | ({
    type: 'markdown';
} & MarkdownProps) | ({
    type: 'document';
} & DocumentViewerProps) | ({
    type: 'video';
} & VideoPlayerProps) | ({
    type: 'audio';
} & AudioPlayerProps);
/** Discriminated props preserve each renderer's API and prevent ambiguous URL/content inference. */
export declare function ContentRenderer(props: ContentRendererProps): import("react").JSX.Element;
