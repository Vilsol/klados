<script lang="ts">
  import Header from './Header.svelte'
  import Sidebar from './Sidebar.svelte'
  import TabBar from './TabBar.svelte'
  import type { Snippet } from 'svelte'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'

  let { children }: { children: Snippet } = $props()

  const basePluginURL = $derived(
    streamingStore.config
      ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins`
      : null
  )
</script>

<a href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-3 focus:py-1.5 focus:bg-bg focus:border focus:border-border focus:rounded focus:text-sm">
  Skip to main content
</a>
<div class="flex flex-col h-full">
  <Header />
  <div class="flex flex-1 overflow-hidden relative">
    <Sidebar />
    <main id="main-content" class="flex flex-col flex-1 overflow-hidden" tabindex="-1">
      <TabBar />
      <div class="flex-1 overflow-auto">
        {@render children()}
      </div>
    </main>
  </div>
  {#if basePluginURL && slotRegistry.getStatusBarWidgets().length > 0}
    <div class="border-t border-border bg-surface flex items-center gap-2 px-3 py-1">
      {#each slotRegistry.getStatusBarWidgets() as widget (widget.id)}
        {#await loadPluginComponent(widget.pluginName, widget.component, basePluginURL) then Cmp}
          {#if Cmp}
            <svelte:component this={Cmp} />
          {/if}
        {/await}
      {/each}
    </div>
  {/if}
</div>
