import { ComponentPropsWithoutRef, ReactElement, ReactNode } from 'react';
import * as Primitive from '@radix-ui/react-dialog';
export interface DialogProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    trigger?: ReactElement;
    title: ReactNode;
    description?: ReactNode;
    footer?: ReactNode;
    className?: string;
    size?: 'sm' | 'md' | 'lg';
    closeLabel?: string;
}
export declare function Dialog({ trigger, title, description, footer, children, className, size, closeLabel, ...props }: DialogProps): import("react").JSX.Element;
export declare const DialogClose: import('react').ForwardRefExoticComponent<Primitive.DialogCloseProps & import('react').RefAttributes<HTMLButtonElement>>;
