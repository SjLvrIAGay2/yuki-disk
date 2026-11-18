import { ReactNode } from 'react';
export interface PlayerControlOptions {
    /** Custom controls by default; native remains available as an explicit fallback. */
    controlsVariant?: 'custom' | 'native';
    playbackRates?: readonly number[];
}
export declare function usePlayer(media: HTMLMediaElement | null): {
    waiting: boolean;
    error: string;
    toggle: () => Promise<void>;
    seek: (value: number) => void;
    volumeTo: (value: number) => void;
    toggleMute: () => void;
    rateTo: (value: number) => void;
    playing: boolean;
    time: number;
    duration: number;
    buffered: number;
    volume: number;
    muted: boolean;
    rate: number;
    ended: boolean;
};
export declare function PlayerIcon({ name, }: {
    name: 'play' | 'pause' | 'replay' | 'volume' | 'mute' | 'expand' | 'collapse' | 'back' | 'forward' | 'music';
}): import("react").JSX.Element;
export declare function PlayerControls({ player, enabled, playbackRates, extra, label, menuContainer, onMenuOpenChange, compact, summary, }: {
    player: ReturnType<typeof usePlayer>;
    enabled: boolean;
    playbackRates?: readonly number[];
    extra?: ReactNode;
    label: string;
    menuContainer?: HTMLElement | null;
    onMenuOpenChange?: (open: boolean) => void;
    compact?: boolean;
    summary?: ReactNode;
}): import("react").JSX.Element;
export declare function VideoControlsExtra({ media, container, onMenuOpenChange, }: {
    media: HTMLVideoElement | null;
    container: HTMLElement | null;
    onMenuOpenChange?: (open: boolean) => void;
}): import("react").JSX.Element;
