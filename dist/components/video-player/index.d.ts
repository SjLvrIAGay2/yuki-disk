import { VideoHTMLAttributes } from 'react';
import { PlayerControlOptions } from '../../internal/player';
import { MediaSource, MediaMetadata, MediaFailure } from '../../internal/media';
export { parseMediaMetadata, formatMediaTime } from '../../internal/media';
export type { MediaSource, MediaMetadata, MediaFailure } from '../../internal/media';
export interface VideoTrack {
    src: string;
    label: string;
    srcLang: string;
    kind?: 'subtitles' | 'captions' | 'descriptions' | 'chapters' | 'metadata';
    default?: boolean;
}
export interface VideoPlayerProps extends Omit<VideoHTMLAttributes<HTMLVideoElement>, 'src'>, PlayerControlOptions {
    src?: MediaSource;
    tracks?: readonly VideoTrack[];
    showMetadata?: boolean;
    onMetadata?: (metadata: MediaMetadata) => void;
    onMediaError?: (error: MediaFailure) => void;
}
export declare const VideoPlayer: import('react').ForwardRefExoticComponent<VideoPlayerProps & import('react').RefAttributes<HTMLVideoElement>>;
