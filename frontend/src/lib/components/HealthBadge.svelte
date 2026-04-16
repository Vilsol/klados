<script lang="ts">
  import { computeHealth, getConditions } from "../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let health = $derived(computeHealth(getConditions(obj)));

  function dotClass(level: string): string {
    switch (level) {
      case "healthy": return "bg-emerald-500";
      case "unhealthy": return "bg-destructive";
      case "progressing": return "bg-amber-500";
      default: return "bg-muted";
    }
  }
</script>

{#if health.level === "unknown"}
  <!-- render nothing when there are no conditions -->
{:else if health.level === "mixed"}
  <span class="text-xs text-muted">{health.reason}</span>
{:else}
  <span
    class="inline-block w-2.5 h-2.5 rounded-full {dotClass(health.level)}"
    title={health.reason}
    aria-label={health.reason}
  ></span>
{/if}
