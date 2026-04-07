type $$ComponentProps = {
    lines: string[];
    eofReached?: boolean;
    eofHistory?: boolean;
    historyLoading?: boolean;
    showTimestamps?: boolean;
    onLoadHistory?: () => void;
    filename?: string;
};
declare const VirtualLogViewer: import("svelte").Component<$$ComponentProps, {
    prependLines: (batch: string[]) => void;
    scrollToLine: (index: number, _align?: "start" | "end") => void;
    scrollToTop: () => void;
    downloadVisible: () => void;
}, "">;
type VirtualLogViewer = ReturnType<typeof VirtualLogViewer>;
export default VirtualLogViewer;
