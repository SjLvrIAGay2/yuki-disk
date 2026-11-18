import { DocumentLoadInfo } from './documents';
export interface PdfOptions {
    password?: string;
    cMapUrl?: string;
    standardFontDataUrl?: string;
    wasmUrl?: string;
}
export interface PdfPreviewProps {
    data: ArrayBuffer;
    filename?: string;
    options?: PdfOptions;
    onLoad?: (info: DocumentLoadInfo) => void;
    onError?: (error: Error) => void;
}
export default function PdfPreview({ data, filename, options, onLoad, onError }: PdfPreviewProps): import("react").JSX.Element;
