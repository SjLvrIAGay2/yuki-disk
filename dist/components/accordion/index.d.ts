import { ComponentPropsWithoutRef } from 'react';
import * as Primitive from '@radix-ui/react-accordion';
export type AccordionProps = ComponentPropsWithoutRef<typeof Primitive.Root>;
export declare const Accordion: import('react').ForwardRefExoticComponent<AccordionProps & import('react').RefAttributes<HTMLDivElement>>;
export declare const AccordionItem: import('react').ForwardRefExoticComponent<Omit<Primitive.AccordionItemProps & import('react').RefAttributes<HTMLDivElement>, "ref"> & import('react').RefAttributes<HTMLDivElement>>;
export declare const AccordionTrigger: import('react').ForwardRefExoticComponent<Omit<Primitive.AccordionTriggerProps & import('react').RefAttributes<HTMLButtonElement>, "ref"> & import('react').RefAttributes<HTMLButtonElement>>;
export declare const AccordionContent: import('react').ForwardRefExoticComponent<Omit<Primitive.AccordionContentProps & import('react').RefAttributes<HTMLDivElement>, "ref"> & import('react').RefAttributes<HTMLDivElement>>;
