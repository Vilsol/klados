<script lang="ts">
  import { X, ChevronDown, ChevronUp, ExternalLink, ScrollText, TerminalSquare, Layers, FileCode } from 'lucide-svelte'
  import { bottomPanelStore, type PanelKind } from '$lib/stores/bottom-panel.svelte'
  import LogsPanel from './panels/LogsPanel.svelte'
  import TerminalPanel from './panels/TerminalPanel.svelte'
  import AggregateLogsPanel from './panels/AggregateLogsPanel.svelte'
  import { YAMLEditor } from '@klados/ui'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import * as SchemaService from '../../../bindings/github.com/Vilsol/klados/internal/services/schemaservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte.js'
  import { unwrapError } from '$lib/utils/async.js'

  const kindLabel: Record<PanelKind, string> = {
    'logs': 'Logs',
    'terminal': 'Terminal',
    'aggregate-logs': 'Logs',
    'yaml': 'YAML',
  }

  const visibleTabs = $derived(bottomPanelStore.visibleTabs)
</script>

{#if bottomPanelStore.hasVisibleTabs}
  <div class="flex flex-col shrink-0 border-t border-border bg-bg" style:height={bottomPanelStore.collapsed ? 'auto' : `${bottomPanelStore.height}px`}>
    <!-- Tab bar -->
    <div class="flex items-center shrink-0 border-b border-border bg-surface">
      <div class="flex items-center overflow-x-auto flex-1" role="tablist">
        {#each visibleTabs as tab (tab.id)}
          <div
            class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs border-r border-border whitespace-nowrap transition-colors select-none cursor-pointer
              {tab.id === bottomPanelStore.activeTabId ? 'bg-bg font-medium text-fg' : 'hover:bg-surface-hover text-muted'}"
            role="tab"
            tabindex="0"
            aria-selected={tab.id === bottomPanelStore.activeTabId}
            onclick={() => bottomPanelStore.setActive(tab.id)}
            onkeydown={(e) => { if (e.key === 'Enter') bottomPanelStore.setActive(tab.id) }}
          >
            {#if tab.kind === 'logs'}
              <ScrollText size={13} />
            {:else if tab.kind === 'terminal'}
              <TerminalSquare size={13} />
            {:else if tab.kind === 'aggregate-logs'}
              <Layers size={13} />
            {:else}
              <FileCode size={13} />
            {/if}
            <span class="truncate max-w-32">{kindLabel[tab.kind]}: {tab.resourceName}</span>
            <button
              onclick={(e) => { e.stopPropagation(); bottomPanelStore.closeTab(tab.id) }}
              class="p-0.5 rounded hover:bg-border transition-colors"
              aria-label="Close tab"
            >
              <X size={11} />
            </button>
          </div>
        {/each}
      </div>
      <button
        onclick={() => bottomPanelStore.toggleCollapsed()}
        class="p-1.5 mx-1 rounded hover:bg-surface-hover text-muted hover:text-fg transition-colors shrink-0"
        aria-label={bottomPanelStore.collapsed ? 'Expand panel' : 'Collapse panel'}
      >
        {#if bottomPanelStore.collapsed}
          <ChevronUp size={14} />
        {:else}
          <ChevronDown size={14} />
        {/if}
      </button>
    </div>

    <!-- Content area -->
    {#if !bottomPanelStore.collapsed}
      <div class="flex-1 overflow-hidden relative">
        {#each visibleTabs as tab (tab.id)}
          <div class="absolute inset-0 overflow-hidden" style:display={tab.id === bottomPanelStore.activeTabId ? 'block' : 'none'}>
            {#if tab.kind === 'logs'}
              <LogsPanel obj={tab.obj} ctxName={tab.ctxName} namespace={tab.namespace} name={tab.name} />
            {:else if tab.kind === 'terminal'}
              <TerminalPanel obj={tab.obj} ctxName={tab.ctxName} namespace={tab.namespace} name={tab.name} />
            {:else if tab.kind === 'aggregate-logs'}
              <AggregateLogsPanel obj={tab.obj} ctxName={tab.ctxName} namespace={tab.namespace} name={tab.name} />
            {:else if tab.kind === 'yaml'}
              <YAMLEditor
                obj={tab.obj}
                ctxName={tab.ctxName}
                gvr={tab.gvr}
                namespace={tab.namespace}
                name={tab.name}
                kind={tab.resourceKind}
                onSave={(ctx: string, g: string, ns: string, parsed: Record<string, any>) => ResourceService.UpdateResource(ctx, g, ns, parsed)}
                onGetResource={(ctx: string, g: string, ns: string, n: string) => ResourceService.GetResource(ctx, g, ns, n)}
                onGetSchema={(ctx: string, g: string, k: string) => SchemaService.GetSchema(ctx, g, k)}
                onNotify={(msg: string, type: 'info' | 'success' | 'error') => {
                  if (type === 'success') notificationStore.success(msg)
                  else if (type === 'error') notificationStore.error(unwrapError(msg))
                  else notificationStore.push(msg, type)
                }}
              />
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}
