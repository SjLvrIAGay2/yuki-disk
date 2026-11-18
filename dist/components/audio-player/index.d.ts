import { AudioHTMLAttributes } from 'react';
import { PlayerControlOptions } from '../../internal/player';
import { MediaSource, MediaMetadata, MediaFailure } from '../../internal/media';
export { parseMediaMetadata, formatMediaTime } from '../../internal/media';
export type { MediaSource, MediaMetadata, MediaFailure } from '../../internal/media';
export interface AudioPlayerProps extends Omit<AudioHTMLAttributes<HTMLAudioElement>, 'src'>, PlayerControlOptions {
    src?: MediaSource;
    description?: string;
    cover?: string;
    showMetadata?: boolean;
    onMetadata?: (metadata: MediaMetadata) => void;
    onMediaError?: (error: MediaFailure) => void;
}
export declare const AudioPlayer: import('react').ForwardRefExoticComponent<AudioPlayerProps & import('react').RefAttributes<HTMLAudioElement>>;
