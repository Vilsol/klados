import { Dialog } from 'bits-ui';
import type { Snippet } from 'svelte';
type $$ComponentProps = {
    open?: boolean;
    title?: string;
    description?: string;
    trigger?: Snippet;
    children?: Snippet;
};
declare const Dialog: import("svelte").Component<$$ComponentProps, {}, "open">;
type Dialog = ReturnType<typeof Dialog>;
export default Dialog;
