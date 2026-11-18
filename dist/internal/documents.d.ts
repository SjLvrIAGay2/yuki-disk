export type DocumentFormat = 'markdown' | 'text' | 'code' | 'pdf' | 'docx';
export type DocumentSource = string | Blob;
export interface DocumentLoadInfo {
    format: DocumentFormat;
    filename?: string;
    size: number;
    pageCount?: number;
    text?: string;
    warnings?: string[];
}
export declare function sourceFilename(source?: DocumentSource): string | undefined;
export declare function inferDocumentFormat(filename?: string, mimeType?: string): DocumentFormat | undefined;
export declare function sourceLanguage(filename?: string): string;
/** Bounded, cancellable reads. Remote requests use the browser's normal CORS policy. */
export declare function readDocument(source: DocumentSource, signal: AbortSignal, maxBytes: number): Promise<{
    data: ArrayBuffer;
    mimeType: string;
}>;
export declare function detectDocumentFormat(data: ArrayBuffer, filename?: string, mimeType?: string): DocumentFormat;
