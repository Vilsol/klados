<script lang="ts">
  let {
    port,
    hostPort,
    protocol = "TCP",
    name,
    onclick,
  }: {
    port: number;
    hostPort?: number;
    protocol?: string;
    name?: string;
    onclick: () => void;
  } = $props();

  const portLabel = $derived(hostPort ? `${hostPort}:${port}` : String(port));
  const titleText = $derived(
    name
      ? `Forward ${name} (${portLabel}/${protocol})`
      : `Forward port ${portLabel}/${protocol}`,
  );
</script>

<button
  type="button"
  {onclick}
  class="inline-flex items-center gap-1.5 text-xs font-mono bg-surface border border-border rounded px-1.5 py-0.5 hover:bg-surface-hover hover:border-accent/50 transition-colors"
  title={titleText}
  aria-label={titleText}
>
  {#if name}<span class="text-muted">{name}</span>{/if}<span>{portLabel}/{protocol}</span>
  <span class="text-muted">↗</span>
</button>
