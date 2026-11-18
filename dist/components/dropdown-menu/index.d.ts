import { ComponentPropsWithoutRef, ReactElement, ReactNode } from 'react';
import * as Primitive from '@radix-ui/react-dropdown-menu';
export interface DropdownMenuProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    trigger: ReactElement;
    align?: 'start' | 'center' | 'end';
    className?: string;
}
export declare function DropdownMenu({ trigger, children, align, className, ...props }: DropdownMenuProps): import("react").JSX.Element;
export interface DropdownMenuItemProps extends ComponentPropsWithoutRef<typeof Primitive.Item> {
    destructive?: boolean;
    icon?: ReactNode;
    shortcut?: string;
}
export declare const DropdownMenuItem: import('react').ForwardRefExoticComponent<DropdownMenuItemProps & import('react').RefAttributes<HTMLDivElement>>;
export declare function DropdownMenuLabel({ className, ...props }: ComponentPropsWithoutRef<typeof Primitive.Label>): import("react").JSX.Element;
export declare function DropdownMenuSeparator({ className, ...props }: ComponentPropsWithoutRef<typeof Primitive.Separator>): import("react").JSX.Element;
