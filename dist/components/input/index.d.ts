import { InputHTMLAttributes, ReactNode } from 'react';
export interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size' | 'prefix'> {
    label?: ReactNode;
    description?: ReactNode;
    error?: ReactNode;
    startIcon?: ReactNode;
    endIcon?: ReactNode;
    size?: 'sm' | 'md' | 'lg';
}
export declare const Input: import('react').ForwardRefExoticComponent<InputProps & import('react').RefAttributes<HTMLInputElement>>;
