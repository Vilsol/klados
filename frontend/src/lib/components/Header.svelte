<script lang="ts">
  import {Lock, LockOpen} from "lucide-svelte";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import ConnectionIndicator from "./ConnectionIndicator.svelte";
  import {Combobox} from "@klados/ui";
  import {push} from "svelte-spa-router";
  import {slotRegistry} from "$lib/plugins/slots.svelte.js";
  import {loadPluginComponent} from "$lib/plugins/loader.js";
  import {streamingStore} from "$lib/stores/streaming.svelte.js";
  import {Events} from "@wailsio/runtime";
  import {ListActive} from "../../../bindings/github.com/Vilsol/klados/internal/services/drainservice.js";

  const ctx = $derived(clusterStore.activeContext);
  const selected = $derived(ctx ? clusterStore.getSelectedNamespaces(ctx) : []);
  const nsOptions = $derived((ctx ? clusterStore.getNamespaces(ctx) : []).map((ns) => ({value: ns, label: ns})));

  function onNamespaceChange(namespaces: string[]) {
    if (ctx) {
      clusterStore.setNamespaces(ctx, namespaces);
    }
  }

  let activeDrains = $state<string[]>([]);

  $effect(() => {
    const currentCtx = ctx;
    if (!currentCtx) {
      activeDrains = [];
      return;
    }

    ListActive(currentCtx).then((nodes: string[]) => {
      activeDrains = nodes ?? [];
    });

    const unsub = Events.On(`drain:${currentCtx}:updated`, () => {
      ListActive(currentCtx).then((nodes: string[]) => {
        activeDrains = nodes ?? [];
      });
    });
    return unsub;
  });

  const basePluginURL = $derived(
    streamingStore.config ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins` : null,
  );
</script>

<header class="flex items-center h-12 px-4 border-b border-border bg-surface shrink-0 gap-4">
  <span class="font-semibold text-sm tracking-wide">Klados</span>

  <div class="flex items-center gap-2 ml-4">
    {#if ctx}
      <ConnectionIndicator status={clusterStore.connectionStatus[ctx] ?? 'disconnected'} clusterName={ctx} />
      <button type="button" onclick={() => push('/clusters')} class="text-sm font-medium hover:underline">{ctx}</button>
    {:else}
      <button type="button" onclick={() => push('/clusters')} class="text-sm text-muted hover:underline">No cluster selected</button>
    {/if}
  </div>

  {#if ctx && nsOptions.length > 0}
    <div class="w-48 ml-2">
      <Combobox
        type="multiple"
        options={nsOptions}
        value={selected}
        allLabel="All Namespaces"
        placeholder="All Namespaces"
        searchPlaceholder="Search namespaces…"
        size="xs"
        onValueChange={onNamespaceChange}
      />
    </div>
  {/if}

  {#if basePluginURL}
    {#each slotRegistry.getHeaderWidgets() as widget (widget.id)}
      {#await loadPluginComponent(widget.pluginName, widget.component, basePluginURL) then Cmp}
        {#if Cmp}
          <Cmp />
        {/if}
      {/await}
    {/each}
  {/if}

  {#if activeDrains.length > 0}
    <div class="flex items-center gap-1.5 text-xs px-2 py-1 rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">
      <span class="inline-block w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
      Draining {activeDrains.length} node{activeDrains.length === 1 ? '' : 's'}
    </div>
  {/if}

  <div class="ml-auto flex items-center gap-2">
    <button
      type="button"
      onclick={() => clusterStore.setReadOnly(!clusterStore.isReadOnly)}
      title={clusterStore.isReadOnly ? 'Read-only mode (click to disable)' : 'Click to enable read-only mode'}
      aria-label="Toggle read-only mode"
      class="flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors {clusterStore.isReadOnly ? 'bg-amber-500/20 text-amber-400 border border-amber-500/30' : 'hover:bg-surface-hover text-muted'}"
    >
      {#if clusterStore.isReadOnly}
        <Lock size={13} />
        <span>Read-only</span>
      {:else}
        <LockOpen size={13} />
      {/if}
    </button>
  </div>
</header>
