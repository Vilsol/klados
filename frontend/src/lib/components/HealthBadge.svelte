<script lang="ts">
  import { computeHealth } from "../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let health = $derived(computeHealth(obj));

  function dotClass(level: string): string {
    switch (level) {
      case "healthy": return "bg-emerald-500";
      case "unhealthy": return "bg-destructive";
      case "progressing": return "bg-amber-500";
      default: return "bg-muted";
    }
  }
</script>

{#if health.level !== "unknown"}
  <span
    class="inline-block w-2.5 h-2.5 rounded-full shrink-0 {dotClass(health.level)}"
    title={health.reason}
    aria-label={health.reason}
  ></span>
{/if}
