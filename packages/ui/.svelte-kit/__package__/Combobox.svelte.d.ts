import { Combobox } from 'bits-ui';
type BaseOption = {
    value: string;
    label: string;
};
type SingleProps = {
    type?: 'single';
    options: BaseOption[];
    value: string;
    placeholder?: string;
    searchPlaceholder?: string;
    emptyMessage?: string;
    size?: 'xs' | 'sm';
    disabled?: boolean;
};
type MultipleProps = {
    type: 'multiple';
    options: BaseOption[];
    value: string[];
    placeholder?: string;
    searchPlaceholder?: string;
    emptyMessage?: string;
    size?: 'xs' | 'sm';
    disabled?: boolean;
};
type Props = SingleProps | MultipleProps;
declare const Combobox: import("svelte").Component<Props, {}, "value">;
type Combobox = ReturnType<typeof Combobox>;
export default Combobox;
