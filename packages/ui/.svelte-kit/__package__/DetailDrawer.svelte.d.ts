import type { Snippet } from 'svelte';
type $$ComponentProps = {
    item: Record<string, any>;
    ctxName: string;
    gvr: string;
    onclose: () => void;
    onFetchResource?: (ctx: string, gvr: string, ns: string, name: string) => Promise<Record<string, any> | null>;
    children: Snippet<[{
        obj: Record<string, any>;
        onrefresh: () => void;
    }]>;
};
declare const DetailDrawer: import("svelte").Component<$$ComponentProps, {}, "">;
type DetailDrawer = ReturnType<typeof DetailDrawer>;
export default DetailDrawer;
