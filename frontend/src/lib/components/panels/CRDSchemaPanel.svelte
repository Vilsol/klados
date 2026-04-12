<script lang="ts">
  import {onMount, onDestroy} from "svelte";
  import {Combobox} from "@klados/ui";
  import {EditorView, keymap, lineNumbers} from "@codemirror/view";
  import {syntaxHighlighting, foldGutter, foldKeymap} from "@codemirror/language";
  import {oneDarkHighlightStyle} from "@codemirror/theme-one-dark";
  import {json as jsonLang} from "@codemirror/lang-json";

  const {obj}: {obj: any} = $props();

  const servedVersions = $derived(((obj.spec?.versions ?? []) as any[]).filter((v) => v.served !== false));

  let selectedVersion = $state("");
  let container: HTMLDivElement;
  let view: EditorView | undefined;

  $effect(() => {
    if (servedVersions.length > 0 && !selectedVersion) {
      selectedVersion = servedVersions[0].name;
    }
  });

  function getContent(): string {
    const v = servedVersions.find((v) => v.name === selectedVersion);
    const schema = v?.schema?.openAPIV3Schema;
    if (!schema) {
      return "// No schema defined for this version";
    }
    return JSON.stringify(schema, null, 2);
  }

  const editorTheme = EditorView.theme({
    "&": {height: "100%", fontSize: "12.5px", backgroundColor: "var(--color-bg)", color: "var(--color-fg)"},
    ".cm-content": {
      padding: "4px 0",
      fontFamily: '"JetBrains Mono", "Fira Code", ui-monospace, monospace',
      caretColor: "var(--color-accent)",
    },
    ".cm-gutters": {
      backgroundColor: "var(--color-surface)",
      color: "var(--color-muted)",
      borderRight: "1px solid var(--color-border)",
      minWidth: "3rem",
    },
    ".cm-lineNumbers .cm-gutterElement": {padding: "0 8px", minWidth: "2.5rem"},
    ".cm-foldGutter .cm-gutterElement": {padding: "0 2px", cursor: "pointer"},
    ".cm-activeLineGutter": {backgroundColor: "var(--color-surface-hover)", color: "var(--color-fg)"},
    ".cm-activeLine": {backgroundColor: "color-mix(in srgb, var(--color-surface-hover) 60%, transparent)"},
    ".cm-foldPlaceholder": {
      backgroundColor: "var(--color-surface)",
      border: "1px solid var(--color-border)",
      color: "var(--color-muted)",
      borderRadius: "3px",
      padding: "0 4px",
    },
    ".cm-scroller": {overflow: "auto", lineHeight: "1.6"},
  });

  onMount(() => {
    view = new EditorView({
      doc: getContent(),
      extensions: [
        lineNumbers(),
        foldGutter(),
        syntaxHighlighting(oneDarkHighlightStyle),
        jsonLang(),
        keymap.of([...foldKeymap]),
        EditorView.editable.of(false),
        EditorView.lineWrapping,
        editorTheme,
      ],
      parent: container,
    });
  });

  $effect(() => {
    const version = selectedVersion;
    if (view && version) {
      const content = getContent();
      view.dispatch({changes: {from: 0, to: view.state.doc.length, insert: content}});
    }
  });

  onDestroy(() => {
    view?.destroy();
  });
</script>

<div class="h-full flex flex-col">
  {#if servedVersions.length > 1}
    <div class="flex items-center gap-2 px-3 py-2 border-b border-border bg-surface">
      <span class="text-sm text-muted">Version:</span>
      <div class="w-32">
        <Combobox
          bind:value={selectedVersion}
          options={servedVersions.map((v) => ({ value: v.name, label: v.name }))}
          placeholder="Version"
        />
      </div>
    </div>
  {/if}
  <div class="flex-1 overflow-hidden" bind:this={container}></div>
</div>
