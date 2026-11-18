import { ComponentPropsWithoutRef, ReactNode } from 'react';
import * as Primitive from '@radix-ui/react-radio-group';
export interface RadioOption {
    value: string;
    label: ReactNode;
    disabled?: boolean;
    description?: ReactNode;
}
export interface RadioGroupProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    options: RadioOption[];
    label?: string;
}
export declare const RadioGroup: import('react').ForwardRefExoticComponent<RadioGroupProps & import('react').RefAttributes<HTMLDivElement>>;
