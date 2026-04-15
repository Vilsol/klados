<script lang="ts">
  import type {Severity} from "./event-types";
  let {severity}: {severity: Severity} = $props();
</script>

<span
  class="px-1.5 py-0.5 rounded text-xs font-medium
  {severity === 'Warning' ? 'bg-destructive/15 text-destructive' : 'bg-accent/15 text-accent'}"
>
  {severity}
</span>
