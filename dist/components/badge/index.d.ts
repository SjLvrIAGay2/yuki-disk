import { HTMLAttributes } from 'react';
export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
    variant?: 'neutral' | 'primary' | 'success' | 'warning' | 'danger';
    dot?: boolean;
}
export declare function Badge({ variant, dot, className, children, ...props }: BadgeProps): import("react").JSX.Element;
