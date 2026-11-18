import { ComponentPropsWithoutRef } from 'react';
import * as Primitive from '@radix-ui/react-tabs';
export interface TabsProps extends ComponentPropsWithoutRef<typeof Primitive.Root> {
    variant?: 'line' | 'segmented';
}
export declare const Tabs: import('react').ForwardRefExoticComponent<TabsProps & import('react').RefAttributes<HTMLDivElement>>;
export declare const TabsList: import('react').ForwardRefExoticComponent<Omit<Primitive.TabsListProps & import('react').RefAttributes<HTMLDivElement>, "ref"> & import('react').RefAttributes<HTMLDivElement>>;
export declare const TabsTrigger: import('react').ForwardRefExoticComponent<Omit<Primitive.TabsTriggerProps & import('react').RefAttributes<HTMLButtonElement>, "ref"> & import('react').RefAttributes<HTMLButtonElement>>;
export declare const TabsContent: import('react').ForwardRefExoticComponent<Omit<Primitive.TabsContentProps & import('react').RefAttributes<HTMLDivElement>, "ref"> & import('react').RefAttributes<HTMLDivElement>>;
