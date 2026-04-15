<script lang="ts">
  import { Columns3, X } from 'lucide-svelte'
  import { Combobox } from '@klados/ui'

  type Props = {
    showWarning: boolean
    showNormal: boolean
    onSeverityChange: (next: { showWarning: boolean; showNormal: boolean }) => void

    availableKinds: string[]
    selectedKinds: string[]
    onKindsChange: (v: string[]) => void

    availableReasons: string[]
    selectedReasons: string[]
    onReasonsChange: (v: string[]) => void

    search: string
    onSearchChange: (v: string) => void

    grouped: boolean
    onGroupedChange: (v: boolean) => void

    paused: boolean
    onJumpToLatest: () => void

    totalCount: number
    warningCount: number
    rangeLabel: string

    columnMenuOpen: boolean
    onColumnMenuToggle: () => void

    timeWindow: { from: number; to: number } | null
    onClearTimeWindow: () => void
  }

  let {
    showWarning,
    showNormal,
    onSeverityChange,
    availableKinds,
    selectedKinds,
    onKindsChange,
    availableReasons,
    selectedReasons,
    onReasonsChange,
    search,
    onSearchChange,
    grouped,
    onGroupedChange,
    paused,
    onJumpToLatest,
    totalCount,
    warningCount,
    rangeLabel,
    columnMenuOpen,
    onColumnMenuToggle,
    timeWindow,
    onClearTimeWindow,
  }: Props = $props()

  function padHHMM(ts: number): string {
    const d = new Date(ts)
    return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  }
</script>

<div class="flex items-center gap-2 px-3 py-2 border-b border-border shrink-0">
  <button
    type="button"
    onclick={() => onSeverityChange({ showWarning: !showWarning, showNormal })}
    class="px-2 py-0.5 rounded-full text-xs border transition-colors {showWarning
      ? 'bg-destructive/15 text-destructive border-destructive/30'
      : 'text-muted border-border'}"
  >
    Warning
  </button>

  <button
    type="button"
    onclick={() => onSeverityChange({ showWarning, showNormal: !showNormal })}
    class="px-2 py-0.5 rounded-full text-xs border transition-colors {showNormal
      ? 'bg-accent/15 text-accent border-accent/30'
      : 'text-muted border-border'}"
  >
    Normal
  </button>

  {#if availableKinds.length > 0}
    <Combobox
      type="multiple"
      options={availableKinds.map((k) => ({ value: k, label: k }))}
      value={selectedKinds}
      placeholder="All kinds"
      onValueChange={onKindsChange}
      size="xs"
    />
  {/if}

  {#if availableReasons.length > 0}
    <Combobox
      type="multiple"
      options={availableReasons.map((r) => ({ value: r, label: r }))}
      value={selectedReasons}
      placeholder="All reasons"
      onValueChange={onReasonsChange}
      size="xs"
    />
  {/if}

  <input
    type="text"
    value={search}
    oninput={(e) => onSearchChange((e.currentTarget as HTMLInputElement).value)}
    placeholder="Search message, reason, object…"
    class="h-6 px-2 text-xs rounded border border-border bg-surface focus:outline-none focus:ring-1 focus:ring-accent"
  />

  {#if timeWindow !== null}
    <button
      type="button"
      data-testid="time-window-chip"
      onclick={onClearTimeWindow}
      class="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs border border-border bg-surface hover:bg-surface-hover transition-colors"
    >
      {padHHMM(timeWindow.from)}–{padHHMM(timeWindow.to)}
      <X size={10} />
    </button>
  {/if}

  <button
    type="button"
    data-testid="grouped-toggle"
    onclick={() => onGroupedChange(!grouped)}
    class="px-2 py-0.5 rounded-full text-xs border transition-colors {grouped
      ? 'bg-accent/15 text-accent border-accent/30'
      : 'border-border text-muted'}"
  >
    Group
  </button>

  {#if paused}
    <span class="text-xs text-muted">Paused</span>
    <button
      type="button"
      data-testid="jump-to-latest"
      onclick={onJumpToLatest}
      class="px-2 py-0.5 rounded text-xs border border-border hover:bg-surface-hover transition-colors"
    >
      Jump to latest
    </button>
  {/if}

  <div class="flex-1"></div>

  <span class="text-xs text-muted">{totalCount} events · {warningCount} warnings ({rangeLabel})</span>

  <button
    type="button"
    onclick={onColumnMenuToggle}
    class="p-1 rounded hover:bg-surface-hover transition-colors"
    title="Manage columns"
    aria-label="Manage columns"
    data-testid="column-menu-button"
  >
    <Columns3 size={14} />
  </button>
</div>
