import { HTMLAttributes, ReactNode } from 'react';
export interface EmptyStateProps extends Omit<HTMLAttributes<HTMLDivElement>, 'title'> {
    title: ReactNode;
    description?: ReactNode;
    icon?: ReactNode;
    action?: ReactNode;
}
export declare function EmptyState({ title, description, icon, action, className, ...props }: EmptyStateProps): import("react").JSX.Element;
