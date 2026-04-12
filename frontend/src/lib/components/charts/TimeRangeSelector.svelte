<script lang="ts">
  interface Props {
    value: number;
    onchange: (v: number) => void;
    hasPrometheus: boolean;
  }

  let {value, onchange, hasPrometheus}: Props = $props();

  const presets = [
    {label: "15m", value: 15},
    {label: "1h", value: 60},
    {label: "6h", value: 360},
    {label: "24h", value: 1440},
    {label: "7d", value: 10_080},
  ];
</script>

{#if hasPrometheus}
  <div class="flex gap-1">
    {#each presets as preset}
      <button
        type="button"
        class="px-2 py-0.5 text-xs rounded transition-colors {value === preset.value
          ? 'bg-accent text-white'
          : 'bg-surface text-muted hover:bg-surface-hover hover:text-fg'}"
        onclick={() => onchange(preset.value)}
      >
        {preset.label}
      </button>
    {/each}
  </div>
{/if}
