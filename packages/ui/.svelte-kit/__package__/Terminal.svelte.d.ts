import { Terminal } from '@xterm/xterm';
interface StreamingConfig {
    port: number;
    token: string;
}
type $$ComponentProps = {
    sessionID: string;
    streamingConfig: StreamingConfig;
    ondisconnect?: () => void;
    useWebGL?: boolean;
    onSetShortcutMode?: (mode: string) => void;
};
declare const Terminal: import("svelte").Component<$$ComponentProps, {}, "">;
type Terminal = ReturnType<typeof Terminal>;
export default Terminal;
