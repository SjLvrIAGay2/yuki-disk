import { HTMLAttributes, ReactNode } from 'react';
export interface AlertProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
    title: ReactNode;
    variant?: 'info' | 'success' | 'warning' | 'danger';
    onClose?: () => void;
    closeLabel?: string;
}
export declare function Alert({ title, variant, onClose, closeLabel, children, className, ...props }: AlertProps): import("react").JSX.Element;
