import { HTMLAttributes } from 'react';
export interface CardProps extends HTMLAttributes<HTMLDivElement> {
    variant?: 'outline' | 'elevated' | 'filled';
}
export declare const Card: import('react').ForwardRefExoticComponent<CardProps & import('react').RefAttributes<HTMLDivElement>>;
export declare function CardHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>): import("react").JSX.Element;
export declare function CardTitle({ className, ...props }: HTMLAttributes<HTMLHeadingElement>): import("react").JSX.Element;
export declare function CardDescription({ className, ...props }: HTMLAttributes<HTMLParagraphElement>): import("react").JSX.Element;
export declare function CardContent({ className, ...props }: HTMLAttributes<HTMLDivElement>): import("react").JSX.Element;
export declare function CardFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>): import("react").JSX.Element;
