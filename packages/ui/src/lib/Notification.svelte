<script lang="ts">
  import { notificationStore, type Toast } from './stores/notification.svelte'
  import { X, ChevronDown } from 'lucide-svelte'

  const typeClass: Record<Toast['type'], string> = {
    info: 'bg-surface border-border text-fg',
    success: 'bg-surface border-accent text-fg',
    error: 'bg-destructive border-destructive text-destructive-fg',
  }

  let expanded = $state<Record<string, boolean>>({})

  function toggleDetails(id: string) {
    expanded[id] = !expanded[id]
  }
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
  {#each notificationStore.notifications as toast (toast.id)}
    <div
      class="flex flex-col gap-1 px-3 py-2 rounded border text-sm shadow-lg pointer-events-auto max-w-sm {typeClass[toast.type]}"
    >
      <div class="flex items-center gap-2">
        <span class="flex-1">{toast.message}</span>
        {#if toast.details}
          <button
            onclick={() => toggleDetails(toast.id)}
            class="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
            title="Toggle details"
            aria-label="{expanded[toast.id] ? 'Collapse' : 'Expand'} details"
          >
            <ChevronDown size={14} class="transition-transform {expanded[toast.id] ? 'rotate-180' : ''}" />
          </button>
        {/if}
        <button
          onclick={() => notificationStore.dismiss(toast.id)}
          class="shrink-0 opacity-60 hover:opacity-100 transition-opacity"
          aria-label="Dismiss notification"
        >
          <X size={14} />
        </button>
      </div>
      {#if toast.details && expanded[toast.id]}
        <pre class="text-xs opacity-75 whitespace-pre-wrap break-all max-h-32 overflow-y-auto font-mono mt-1">{toast.details}</pre>
      {/if}
      {#if toast.actions?.length}
        <div class="flex gap-2 mt-1">
          {#each toast.actions as action}
            <button
              onclick={action.onClick}
              class="text-xs underline opacity-75 hover:opacity-100 transition-opacity"
            >{action.label}</button>
          {/each}
        </div>
      {/if}
    </div>
  {/each}
</div>
