import { ReactNode } from 'react';
export interface ToastOptions {
    title: string;
    description?: string;
    variant?: 'info' | 'success' | 'danger';
    duration?: number;
}
interface ToastContextValue {
    toast: (options: ToastOptions) => string;
    dismiss: (id: string) => void;
}
export interface ToastProviderProps {
    children: ReactNode;
    duration?: number;
    label?: string;
    closeLabel?: string;
}
export declare function ToastProvider({ children, duration, label, closeLabel, }: ToastProviderProps): import("react").JSX.Element;
export declare function useToast(): ToastContextValue;
export {};
