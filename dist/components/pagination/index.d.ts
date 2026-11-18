import { HTMLAttributes } from 'react';
export interface PaginationProps extends Omit<HTMLAttributes<HTMLElement>, 'onChange'> {
    page: number;
    totalPages: number;
    onPageChange: (page: number) => void;
    disabled?: boolean;
}
export declare function Pagination({ page, totalPages, onPageChange, disabled, className, ...props }: PaginationProps): import("react").JSX.Element;
