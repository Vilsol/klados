<script lang="ts">
  import {Eye, EyeOff, Copy, Check} from "lucide-svelte";
  import {SectionHeader} from "@klados/ui";
  import {toggleSet} from "$lib/utils/collections";

  let {obj}: {obj: Record<string, any>} = $props();

  const secretType = $derived<string>(obj.type ?? "Opaque");
  const data = $derived<Record<string, string>>(obj.data ?? {});
  const entries = $derived(Object.entries(data));

  let revealed = $state<Set<string>>(new Set());
  let copied = $state<string | null>(null);

  function decode(b64: string): string {
    try {
      return atob(b64);
    } catch {
      return b64;
    }
  }

  async function copyDecoded(key: string, b64: string) {
    try {
      await navigator.clipboard.writeText(decode(b64));
      copied = key;
      setTimeout(() => {
        copied = null;
      }, 2000);
    } catch {
      // clipboard unavailable
    }
  }
</script>

<div class="flex flex-col gap-4 p-4 overflow-auto h-full">
  <!-- Type badge -->
  <div class="flex items-center gap-2">
    <span class="text-xs text-muted">Type</span>
    <span class="text-xs bg-surface-hover border border-border rounded px-2 py-0.5">{secretType}</span>
  </div>

  {#if entries.length === 0}
    <p class="text-sm text-muted">No data</p>
  {:else}
    <section>
      <div class="flex items-center justify-between mb-2">
        <SectionHeader>Data ({entries.length} {entries.length === 1 ? 'key' : 'keys'})</SectionHeader>
        <span class="text-xs text-muted">Values are base64-encoded</span>
      </div>
      <div class="flex flex-col gap-2">
        {#each entries as [ key, value ]}
          {@const isRevealed = revealed.has(key)}
          {@const decoded = decode(value)}
          <div class="bg-surface border border-border rounded-lg overflow-hidden">
            <div class="flex items-center justify-between px-3 py-1.5 border-b border-border bg-surface-hover">
              <span class="text-xs font-medium font-mono">{key}</span>
              <div class="flex items-center gap-1">
                <button
                  onclick={() => copyDecoded(key, value)}
                  class="p-1 rounded hover:bg-surface transition-colors text-muted hover:text-fg"
                  title="Copy decoded value"
                  aria-label="Copy decoded value for {key}"
                >
                  {#if copied === key}
                    <Check size={12} class="text-green-500" />
                  {:else}
                    <Copy size={12} />
                  {/if}
                </button>
                <button
                  onclick={() => revealed = toggleSet(revealed, key)}
                  class="p-1 rounded hover:bg-surface transition-colors text-muted hover:text-fg"
                  title={isRevealed ? 'Hide value' : 'Reveal value'}
                >
                  {#if isRevealed}
                    <EyeOff size={12} />
                  {:else}
                    <Eye size={12} />
                  {/if}
                </button>
              </div>
            </div>
            <div class="px-3 py-2 text-xs font-mono">
              {#if isRevealed}
                <pre class="whitespace-pre-wrap break-all">{decoded}</pre>
              {:else}
                <span class="text-muted tracking-widest">••••••••</span>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>
