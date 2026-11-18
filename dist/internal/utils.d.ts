import { SVGProps } from 'react';
export declare function cx(...values: (string | false | null | undefined)[]): string;
export declare function Icon({ name, ...props }: SVGProps<SVGSVGElement> & {
    name: 'check' | 'chevron' | 'close' | 'info' | 'warning' | 'search';
}): import("react").JSX.Element;
