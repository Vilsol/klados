<script lang="ts">
  let {
    port,
    protocol = "TCP",
    name,
    onclick,
  }: {
    port: number;
    protocol?: string;
    name?: string;
    onclick: () => void;
  } = $props();
</script>

<button
  type="button"
  {onclick}
  class="inline-flex items-center gap-1.5 text-xs font-mono bg-surface border border-border rounded px-1.5 py-0.5 hover:bg-surface-hover hover:border-accent/50 transition-colors"
  title={name ? `Forward ${name} (${port}/${protocol})` : `Forward port ${port}`}
  aria-label={name ? `Forward ${name} (${port}/${protocol})` : `Forward port ${port}`}
>
  {#if name}<span class="text-muted">{name}</span>{/if}<span>{port}/{protocol}</span>
  <span class="text-muted">↗</span>
</button>
