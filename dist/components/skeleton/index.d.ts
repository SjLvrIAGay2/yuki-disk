import { CSSProperties, HTMLAttributes } from 'react';
export interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
    width?: CSSProperties['width'];
    height?: CSSProperties['height'];
    circle?: boolean;
}
export declare function Skeleton({ width, height, circle, className, style, ...props }: SkeletonProps): import("react").JSX.Element;
