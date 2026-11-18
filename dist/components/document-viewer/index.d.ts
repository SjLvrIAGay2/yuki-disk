import { HTMLAttributes } from 'react';
import { MarkdownProps } from '../markdown';
import { DocumentFormat, DocumentLoadInfo, DocumentSource } from '../../internal/documents';
import { PdfOptions } from '../../internal/pdf-preview';
export type { DocumentFormat, DocumentLoadInfo, DocumentSource } from '../../internal/documents';
export type { PdfOptions } from '../../internal/pdf-preview';
export { inferDocumentFormat } from '../../internal/documents';
export interface DocumentViewerProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children' | 'onLoad' | 'onError'> {
    src?: DocumentSource;
    content?: string;
    format?: DocumentFormat | 'auto';
    filename?: string;
    language?: string;
    maxFileSize?: number;
    showHeader?: boolean;
    pdfOptions?: PdfOptions;
    markdownOptions?: Pick<MarkdownProps, 'allowImages' | 'openLinksInNewTab' | 'codeLineNumbers'>;
    onLoad?: (info: DocumentLoadInfo) => void;
    onError?: (error: Error) => void;
}
export declare function DocumentViewer({ src, content, format, filename: name, language, maxFileSize: requestedLimit, showHeader, pdfOptions, markdownOptions, onLoad, onError, className, ...props }: DocumentViewerProps): import("react").JSX.Element;
