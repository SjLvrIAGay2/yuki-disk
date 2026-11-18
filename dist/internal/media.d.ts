export type MediaSource = string | Blob;
export interface MediaMetadata {
    duration: number | null;
    width?: number;
    height?: number;
    src: string;
    mimeType?: string;
    size?: number;
    filename?: string;
}
export interface MediaFailure {
    code: number;
    message: string;
}
export declare function useMediaSource(source?: MediaSource): string | undefined;
export declare function readMediaMetadata(media: HTMLMediaElement, source?: MediaSource): MediaMetadata;
export declare function mediaFailure(error: MediaError | null): MediaFailure;
export declare function formatMediaTime(seconds: number | null): string;
/** Browser metadata decoding without mounting a player. Does not upload or transcribe media. */
export declare function parseMediaMetadata(source: MediaSource, { kind, signal, timeout, }: {
    kind: 'video' | 'audio';
    signal?: AbortSignal;
    timeout?: number;
}): Promise<MediaMetadata>;
