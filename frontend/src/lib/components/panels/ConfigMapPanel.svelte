<script lang="ts">
  import {CodeBlock, SectionHeader} from "@klados/ui";
  import type {KubernetesResource} from "$lib/types";

  const reShebangOrShell = /^(if|for|while|case|function|export|source|set -)\b/m;
  const reToml = /^\[[a-zA-Z]/m;
  const reYaml = /^[a-zA-Z_-]+:/m;

  let {obj}: {obj: Record<string, KubernetesResource>} = $props();

  const data = $derived<Record<string, string>>(obj.data ?? {});
  const binaryData = $derived<Record<string, string>>(obj.binaryData ?? {});

  const dataEntries = $derived(Object.entries(data));
  const binaryKeys = $derived(Object.keys(binaryData));

  function detectLang(value: string): "yaml" | "json" | "toml" | "shell" | "plain" {
    const trimmed = value.trimStart();
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      return "json";
    }
    if (trimmed.startsWith("#!") || reShebangOrShell.test(trimmed)) {
      return "shell";
    }
    if (reToml.test(trimmed)) {
      return "toml";
    }
    if (reYaml.test(trimmed)) {
      return "yaml";
    }
    return "plain";
  }
</script>

<div class="flex flex-col gap-4 p-4 overflow-auto h-full">
  {#if dataEntries.length === 0 && binaryKeys.length === 0}
    <p class="text-sm text-muted">No data</p>
  {/if}

  {#if dataEntries.length > 0}
    <section>
      <SectionHeader>Data ({dataEntries.length} {dataEntries.length === 1 ? 'key' : 'keys'})</SectionHeader>
      <div class="flex flex-col gap-3">
        {#each dataEntries as [ key, value ]}
          {@const lang = detectLang(value)}
          <div class="bg-surface border border-border rounded-lg overflow-hidden">
            <div class="flex items-center justify-between px-3 py-1.5 border-b border-border bg-surface-hover">
              <span class="text-xs font-medium font-mono">{key}</span>
              <span class="text-xs text-muted">{lang}</span>
            </div>
            <CodeBlock {value} lang={detectLang(value)} />
          </div>
        {/each}
      </div>
    </section>
  {/if}

  {#if binaryKeys.length > 0}
    <section>
      <SectionHeader>Binary Data ({binaryKeys.length} {binaryKeys.length === 1 ? 'key' : 'keys'})</SectionHeader>
      <div class="flex flex-col gap-1">
        {#each binaryKeys as key}
          <div class="flex items-center gap-2 px-3 py-1.5 bg-surface border border-border rounded text-sm">
            <span class="font-mono text-xs">{key}</span>
            <span class="text-muted text-xs">(binary)</span>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>
