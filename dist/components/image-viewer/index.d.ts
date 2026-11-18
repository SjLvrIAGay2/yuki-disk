import { ImgHTMLAttributes } from 'react';
import { MediaSource } from '../../internal/media';
export interface ImageViewerProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'src'> {
    src?: MediaSource;
    alt: string;
}
/** Displays an image without interpreting document markup; owns URLs created from Blobs. */
export declare const ImageViewer: import('react').ForwardRefExoticComponent<ImageViewerProps & import('react').RefAttributes<HTMLImageElement>>;
