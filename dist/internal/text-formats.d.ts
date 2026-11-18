export declare function textDocumentType(filename: string): {
    format: 'text' | 'code' | 'markdown';
    language: string;
} | undefined;
