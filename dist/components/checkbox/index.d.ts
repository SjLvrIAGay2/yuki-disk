import { ComponentPropsWithoutRef, ReactNode } from 'react';
import * as Primitive from '@radix-ui/react-checkbox';
export interface CheckboxProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    label?: ReactNode;
    description?: ReactNode;
}
export declare const Checkbox: import('react').ForwardRefExoticComponent<CheckboxProps & import('react').RefAttributes<HTMLButtonElement>>;
