import { ComponentPropsWithoutRef, ReactNode } from 'react';
import * as Primitive from '@radix-ui/react-switch';
export interface SwitchProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    label?: ReactNode;
    description?: ReactNode;
    size?: 'sm' | 'md';
}
export declare const Switch: import('react').ForwardRefExoticComponent<SwitchProps & import('react').RefAttributes<HTMLButtonElement>>;
