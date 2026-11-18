import { HTMLAttributes } from 'react';
import { Components } from 'react-markdown';
export interface MarkdownProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children'> {
    children: string;
    /** Trusted React component overrides, not HTML strings. */
    components?: Components;
    openLinksInNewTab?: boolean;
    codeLineNumbers?: boolean;
    allowImages?: boolean;
}
export declare function Markdown({ children, components, openLinksInNewTab, codeLineNumbers, allowImages, className, ...props }: MarkdownProps): import("react").JSX.Element;
