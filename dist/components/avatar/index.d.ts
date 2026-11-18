import { HTMLAttributes } from 'react';
export interface AvatarProps extends HTMLAttributes<HTMLSpanElement> {
    src?: string;
    alt: string;
    fallback?: string;
    size?: 'sm' | 'md' | 'lg';
    status?: 'online' | 'offline' | 'busy';
}
export declare function Avatar({ src, alt, fallback, size, status, className, ...props }: AvatarProps): import("react").JSX.Element;
export declare function AvatarGroup({ className, ...props }: HTMLAttributes<HTMLDivElement>): import("react").JSX.Element;
