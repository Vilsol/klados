type $$ComponentProps = {
    label?: string;
    error?: string;
    disabled?: boolean;
    value?: string;
    placeholder?: string;
    type?: string;
    id?: string;
};
declare const Input: import("svelte").Component<$$ComponentProps, {}, "value">;
type Input = ReturnType<typeof Input>;
export default Input;
