import type { Snippet } from 'svelte';
type $$ComponentProps = {
    variant?: 'primary' | 'destructive' | 'ghost' | 'outline';
    disabled?: boolean;
    type?: 'button' | 'submit' | 'reset';
    onclick?: () => void;
    children?: Snippet;
};
declare const Button: import("svelte").Component<$$ComponentProps, {}, "">;
type Button = ReturnType<typeof Button>;
export default Button;
