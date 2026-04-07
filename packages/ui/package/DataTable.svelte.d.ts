import type { Snippet } from 'svelte';
type $$ComponentProps = {
    columns: {
        label: string;
        width?: string;
    }[];
    items: any[];
    row: Snippet<[item: any]>;
    sticky?: boolean;
};
declare const DataTable: import("svelte").Component<$$ComponentProps, {}, "">;
type DataTable = ReturnType<typeof DataTable>;
export default DataTable;
