import { ComponentPropsWithoutRef } from 'react';
import * as Primitive from '@radix-ui/react-slider';
export interface SliderProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    label?: string;
    showValue?: boolean;
    formatValue?: (value: number) => string;
    thumbLabels?: string[];
}
export declare const Slider: import('react').ForwardRefExoticComponent<SliderProps & import('react').RefAttributes<HTMLSpanElement>>;
