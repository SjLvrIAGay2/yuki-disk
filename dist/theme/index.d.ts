import { CSSProperties, HTMLAttributes } from 'react';
export type Theme = 'light' | 'dark' | 'system';
export type ThemeTokens = CSSProperties & {
    [key: `--yk-${string}`]: string | number;
};
export interface ThemeProviderProps extends HTMLAttributes<HTMLDivElement> {
    theme?: Theme;
    tokens?: ThemeTokens;
}
/** Scoped tokens also follow portaled dialogs, menus, selects, tooltips and toasts. */
export declare function ThemeProvider({ theme, tokens, className, style, children, ...props }: ThemeProviderProps): import("react").JSX.Element;
export declare function useTheme(): {
    theme: "light" | "dark";
    tokens?: ThemeTokens;
};
