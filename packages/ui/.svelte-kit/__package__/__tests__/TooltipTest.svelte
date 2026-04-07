<script lang="ts">
  import Tooltip from '../Tooltip.svelte'
</script>

<Tooltip content="Hello">
  {#snippet trigger(props)}
    <button {...props}>Hover me</button>
  {/snippet}
</Tooltip>
