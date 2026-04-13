<script lang="ts">
  import {Dialog} from "bits-ui";
  import {shortcutStore} from "$lib/stores/shortcuts.svelte";

  let {open = $bindable(false)}: {open?: boolean} = $props();

  const CATEGORY_ORDER = ["General", "Navigation", "Resources"];

  const grouped = $derived.by(() => {
    const all = shortcutStore.getAll().filter((s) => !s.hidden);
    const groups = new Map<string, typeof all>();
    for (const s of all) {
      const cat = s.category ?? "Other";
      const list = groups.get(cat);
      if (list) {
        list.push(s);
      } else {
        groups.set(cat, [s]);
      }
    }
    // Sort categories by predefined order, unknown categories at the end
    const sorted = [...groups.entries()].sort(([a], [b]) => {
      const ai = CATEGORY_ORDER.indexOf(a);
      const bi = CATEGORY_ORDER.indexOf(b);
      return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi);
    });
    return sorted;
  });

  function formatKeys(keys: string): string[] {
    return keys.split("+");
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[32rem] max-w-[90vw] max-h-[80vh] overflow-y-auto"
    >
      <Dialog.Title class="text-base font-semibold mb-4">Keyboard Shortcuts</Dialog.Title>

      {#each grouped as [category, shortcuts]}
        <div class="mb-4 last:mb-0">
          <h3 class="text-xs font-medium text-muted uppercase tracking-wide mb-2">{category}</h3>
          <div class="space-y-1.5">
            {#each shortcuts as shortcut}
              {@const parts = formatKeys(shortcutStore.getEffectiveKeys(shortcut))}
              <div class="flex items-center justify-between py-1">
                <span class="text-sm text-fg">{shortcut.description}</span>
                <span class="flex items-center gap-0.5 shrink-0 ml-4">
                  {#each parts as part, i}
                    {#if i > 0}
                      <span class="text-muted text-xs">+</span>
                    {/if}
                    <kbd
                      class="inline-flex items-center justify-center min-w-[1.5rem] px-1.5 py-0.5 text-xs font-medium rounded border border-border bg-bg text-muted"
                    >
                      {part}
                    </kbd>
                  {/each}
                </span>
              </div>
            {/each}
          </div>
        </div>
      {/each}

      <Dialog.Close
        class="absolute top-4 right-4 text-muted hover:text-fg transition-colors text-lg leading-none"
        aria-label="Close"
      >
        &times;
      </Dialog.Close>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
