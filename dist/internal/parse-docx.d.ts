export declare function parseDocx(data: ArrayBuffer, idPrefix: string): Promise<{
    html: string;
    text: string;
    warnings: string[];
}>;
