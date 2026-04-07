interface StreamingConfig {
    port: number;
    token: string;
}
type $$ComponentProps = {
    streamID: string;
    streamingConfig: StreamingConfig;
    showTimestamps?: boolean;
    filename?: string;
    scrollToTopOnLoad?: boolean;
};
declare const LogViewer: import("svelte").Component<$$ComponentProps, {
    downloadVisible: () => void;
    scrollToTop: () => void;
    loadHistory: () => void;
}, "">;
type LogViewer = ReturnType<typeof LogViewer>;
export default LogViewer;
