/** Hide only idle fullscreen chrome, keeping keyboard and active pointer interactions usable. */
export declare function useVideoControlsVisibility(container: HTMLElement | null, canHide: boolean): {
    fullscreen: boolean;
    hidden: boolean;
};
