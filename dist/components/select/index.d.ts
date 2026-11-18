import { ComponentPropsWithoutRef } from 'react';
import * as Primitive from '@radix-ui/react-select';
export interface SelectOption {
    value: string;
    label: string;
    disabled?: boolean;
}
export interface SelectProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    options: SelectOption[];
    label?: string;
    placeholder?: string;
    error?: string;
    description?: string;
    id?: string;
    className?: string;
    /** Mount the menu within this element, for example a fullscreen player. */
    portalContainer?: HTMLElement | null;
    'aria-label'?: string;
    'aria-describedby'?: string;
}
export declare const Select: import('react').ForwardRefExoticComponent<SelectProps & import('react').RefAttributes<HTMLButtonElement>>;
