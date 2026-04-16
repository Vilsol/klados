<script lang="ts">
  import {
    getLabels,
    getAnnotations,
    LAST_APPLIED_ANNOTATION,
  } from "../../kubernetes/metadata";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let labels = $derived(getLabels(obj));
  let annotations = $derived.by(() => {
    const all = getAnnotations(obj);
    const { [LAST_APPLIED_ANNOTATION]: _ignored, ...rest } = all;
    return rest;
  });
  let expanded = $state<Record<string, boolean>>({});

  function toggle(key: string) { expanded[key] = !expanded[key]; }
  function isLong(v: string) { return v.length > 120; }
</script>

<div class="p-4 space-y-6 text-sm">
  <section>
    <h3 class="font-semibold mb-2">Labels</h3>
    {#if Object.keys(labels).length === 0}
      <div class="text-muted">No labels.</div>
    {:else}
      <div class="flex flex-wrap gap-2">
        {#each Object.entries(labels) as [k, v]}
          <span class="rounded bg-surface border border-border px-2 py-0.5 font-mono text-xs">
            {k}={v}
          </span>
        {/each}
      </div>
    {/if}
  </section>

  <section>
    <h3 class="font-semibold mb-2">Annotations</h3>
    {#if Object.keys(annotations).length === 0}
      <div class="text-muted">No annotations.</div>
    {:else}
      <table class="w-full text-xs font-mono">
        <tbody>
          {#each Object.entries(annotations) as [k, v]}
            <tr class="border-b border-border">
              <td class="p-2 align-top w-64 break-all">{k}</td>
              <td class="p-2 align-top break-all">
                {#if isLong(v)}
                  {#if expanded[k]}
                    <span class="whitespace-pre-wrap">{v}</span>
                    <button class="ml-2 text-accent text-xs" onclick={() => toggle(k)}>collapse</button>
                  {:else}
                    <span>{v.slice(0, 120)}…</span>
                    <button class="ml-2 text-accent text-xs" onclick={() => toggle(k)}>expand</button>
                  {/if}
                {:else}
                  <span>{v}</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>
</div>
