import type { Snippet } from 'svelte';
type $$ComponentProps = {
    status: 'True' | 'False' | 'Unknown' | 'Warning' | 'Normal' | boolean;
    mode?: 'text' | 'pill';
    children?: Snippet;
};
declare const StatusBadge: import("svelte").Component<$$ComponentProps, {}, "">;
type StatusBadge = ReturnType<typeof StatusBadge>;
export default StatusBadge;
