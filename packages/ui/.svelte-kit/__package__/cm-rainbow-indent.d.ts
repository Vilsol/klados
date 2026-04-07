import { EditorView, ViewPlugin, type DecorationSet, type ViewUpdate } from '@codemirror/view';
export declare const rainbowIndentTheme: import("@codemirror/state").Extension;
declare class RainbowIndentPlugin {
    decorations: DecorationSet;
    constructor(view: EditorView);
    update(update: ViewUpdate): void;
    build(view: EditorView): DecorationSet;
}
export declare const rainbowIndent: ViewPlugin<RainbowIndentPlugin, undefined>;
export {};
