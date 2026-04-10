<script lang="ts">
  import type { Snippet } from 'svelte'
  import { push } from 'svelte-spa-router'
  import { ArrowLeft } from 'lucide-svelte'
  import SettingsSidebar from './SettingsSidebar.svelte'

  interface Props {
    active: string
    children: Snippet
  }

  let { active, children }: Props = $props()
</script>

<div class="flex flex-col h-full bg-bg">
  <div class="flex items-center gap-2 px-4 py-3 border-b border-border shrink-0">
    <button
      onclick={() => push('/')}
      class="p-1 rounded hover:bg-surface-hover transition-colors text-muted hover:text-fg"
      aria-label="Back"
    >
      <ArrowLeft size={16} />
    </button>
    <span class="text-sm font-semibold text-fg">Settings</span>
  </div>

  <div class="flex flex-1 overflow-hidden">
    <SettingsSidebar {active} />
    <main class="flex-1 overflow-y-auto p-6">
      {@render children()}
    </main>
  </div>
</div>
