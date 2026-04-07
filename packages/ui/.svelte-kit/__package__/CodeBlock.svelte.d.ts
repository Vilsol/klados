type $$ComponentProps = {
    value: string;
    lang: 'yaml' | 'json' | 'toml' | 'shell' | 'plain';
};
declare const CodeBlock: import("svelte").Component<$$ComponentProps, {}, "">;
type CodeBlock = ReturnType<typeof CodeBlock>;
export default CodeBlock;
