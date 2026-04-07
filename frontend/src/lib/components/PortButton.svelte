<script lang="ts">
  let {
    port,
    protocol = 'TCP',
    onclick,
  }: {
    port: number
    protocol?: string
    onclick: () => void
  } = $props()
</script>

<button
  {onclick}
  class="text-xs font-mono bg-surface border border-border rounded px-1.5 py-0.5 hover:bg-surface-hover hover:border-accent/50 transition-colors"
  title="Forward port {port}"
  aria-label="Forward port {port}"
>
  {port}/{protocol} ↗
</button>
