type $$ComponentProps = {
    options: {
        value: string;
        label: string;
    }[];
    value: string;
    size?: 'xs' | 'sm';
};
declare const Select: import("svelte").Component<$$ComponentProps, {}, "value">;
type Select = ReturnType<typeof Select>;
export default Select;
