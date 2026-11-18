import { ComponentPropsWithoutRef, ReactElement, ReactNode } from 'react';
import * as Primitive from '@radix-ui/react-tooltip';
export interface TooltipProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    children: ReactElement;
    content: ReactNode;
    side?: 'top' | 'right' | 'bottom' | 'left';
}
export declare function Tooltip({ children, content, side, delayDuration, ...props }: TooltipProps): import("react").JSX.Element;
