type $$ComponentProps = {
    obj: Record<string, any>;
    ctxName: string;
    gvr: string;
    namespace: string;
    name: string;
    kind?: string;
    onrefresh?: () => void;
    onSave?: (ctx: string, gvr: string, ns: string, parsed: Record<string, any>) => Promise<Record<string, any> | null>;
    onGetResource?: (ctx: string, gvr: string, ns: string, name: string) => Promise<Record<string, any> | null>;
    onGetSchema?: (ctx: string, gvr: string, kind: string) => Promise<Record<string, any>>;
    onNotify?: (msg: string, type: 'info' | 'success' | 'error') => void;
    onSetEditorMode?: (mode: string) => void;
    onOpenUrl?: (url: string) => void;
};
declare const YAMLEditor: import("svelte").Component<$$ComponentProps, {}, "obj">;
type YAMLEditor = ReturnType<typeof YAMLEditor>;
export default YAMLEditor;
