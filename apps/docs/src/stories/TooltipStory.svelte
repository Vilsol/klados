<script lang="ts">
  import { Tooltip, Button } from '@klados/ui'

  let {
    content = 'This is a tooltip',
    triggerLabel = 'Hover me',
  }: {
    content?: string
    triggerLabel?: string
  } = $props()
</script>

<div class="flex items-center justify-center p-8">
  <Tooltip {content}>
    {#snippet trigger(props)}
      <Button {...props}>{triggerLabel}</Button>
    {/snippet}
  </Tooltip>
</div>
