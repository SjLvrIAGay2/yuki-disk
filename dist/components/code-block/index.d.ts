import { CSSProperties, HTMLAttributes } from 'react';
export interface CodeBlockProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children' | 'onCopy'> {
    code: string;
    language?: string;
    filename?: string;
    showLineNumbers?: boolean;
    startLine?: number;
    highlightLines?: readonly number[];
    wrap?: boolean;
    maxHeight?: CSSProperties['maxHeight'];
    copyable?: boolean;
    onCopy?: (code: string) => void;
    onCopyError?: (error: Error) => void;
}
export declare function CodeBlock({ code, language, filename, showLineNumbers, startLine, highlightLines, wrap, maxHeight, copyable, onCopy, onCopyError, className, ...props }: CodeBlockProps): import("react").JSX.Element;
