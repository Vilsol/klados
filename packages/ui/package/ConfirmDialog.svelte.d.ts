type $$ComponentProps = {
    open: boolean;
    title?: string;
    message?: string;
    confirmLabel?: string;
    onconfirm: () => void;
};
declare const ConfirmDialog: import("svelte").Component<$$ComponentProps, {}, "open">;
type ConfirmDialog = ReturnType<typeof ConfirmDialog>;
export default ConfirmDialog;
