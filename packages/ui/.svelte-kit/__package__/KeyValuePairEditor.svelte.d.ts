type $$ComponentProps = {
    pairs: [string, string][];
    addLabel?: string;
    keyPlaceholder?: string;
    valuePlaceholder?: string;
};
declare const KeyValuePairEditor: import("svelte").Component<$$ComponentProps, {}, "pairs">;
type KeyValuePairEditor = ReturnType<typeof KeyValuePairEditor>;
export default KeyValuePairEditor;
