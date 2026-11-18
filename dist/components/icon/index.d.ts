import { SVGProps } from 'react';
declare const paths: {
    readonly folder: "M3 7h6l2 2h10v11H3z M3 7V4h6l2 3";
    readonly image: "M3 4h18v16H3z M3 17l6-6 4 4 3-3 5 5 M16 8h.01";
    readonly text: "M5 3h10l4 4v14H5z M8 10h8 M8 14h8 M8 18h5";
    readonly code: "M8 7l-5 5 5 5 M16 7l5 5-5 5 M14 4l-4 16";
    readonly pdf: "M5 3h10l4 4v14H5z M14 3v5h5 M8 12h8 M8 16h5";
    readonly audio: "M9 17V5l11-2v12 M9 8l11-2 M9 17a3 3 0 1 1-3-3h3 M20 15a3 3 0 1 1-3-3h3";
    readonly video: "M3 4h18v16H3z M10 8l6 4-6 4z";
    readonly archive: "M5 3h14v18H5z M10 3h4 M10 6h4 M10 9h4 M11 13h3v4h-3z";
    readonly document: "M5 3h10l4 4v14H5z M8 10l2 7 2-5 2 5 2-7";
    readonly spreadsheet: "M4 3h16v18H4z M4 9h16 M4 15h16 M10 9v12";
    readonly presentation: "M3 4h18v13H3z M12 17v4 M8 21h8 M8 8v5 M12 10v3 M16 7v6";
    readonly file: "M5 3h10l4 4v14H5z M14 3v5h5";
    readonly menu: "M4 6h16 M4 12h16 M4 18h16";
    readonly close: "M6 6l12 12 M18 6L6 18";
    readonly download: "M12 3v12 M7 10l5 5 5-5 M4 16v5h16v-5";
};
export type IconName = keyof typeof paths;
export declare const iconNames: IconName[];
export interface IconProps extends Omit<SVGProps<SVGSVGElement>, 'name' | 'children'> {
    name: IconName;
    size?: number | string;
    label?: string;
}
/** Icons inherit currentColor. Omit label when adjacent text provides the meaning. */
export declare function Icon({ name, size, label, className, ...props }: IconProps): import("react").JSX.Element;
export {};
