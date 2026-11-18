import { Component, ReactNode } from 'react';
export declare class RenderBoundary extends Component<{
    children: ReactNode;
    onError?: (error: Error) => void;
}, {
    error?: Error;
}> {
    state: {
        error?: Error;
    };
    static getDerivedStateFromError(error: Error): {
        error: Error;
    };
    componentDidCatch(error: Error): void;
    render(): string | number | bigint | boolean | import("react").JSX.Element | Iterable<ReactNode> | Promise<string | number | bigint | boolean | import('react').ReactPortal | import('react').ReactElement<unknown, string | import('react').JSXElementConstructor<any>> | Iterable<ReactNode> | null | undefined> | null | undefined;
}
