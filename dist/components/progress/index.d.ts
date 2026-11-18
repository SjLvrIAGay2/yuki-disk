import { HTMLAttributes } from 'react';
export interface ProgressProps extends HTMLAttributes<HTMLDivElement> {
    value?: number | null;
    max?: number;
    label?: string;
    showValue?: boolean;
    size?: 'sm' | 'md';
}
export declare function Progress({ value, max, label, showValue, size, className, ...props }: ProgressProps): import("react").JSX.Element;
