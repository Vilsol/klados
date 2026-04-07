<script lang="ts">
  import { ConfirmDialog, Button } from '@klados/ui'

  let {
    title = 'Confirm',
    message = 'Are you sure?',
    confirmLabel = 'Confirm',
    isDestructive = false,
  }: {
    title?: string
    message?: string
    confirmLabel?: string
    isDestructive?: boolean
  } = $props()

  let open = $state(false)
  let lastAction = $state('')
</script>

<div class="flex flex-col items-start gap-4 p-8">
  <Button variant={isDestructive ? 'destructive' : 'primary'} onclick={() => (open = true)}>
    {confirmLabel}
  </Button>
  {#if lastAction}
    <p class="text-sm text-muted">Last action: {lastAction}</p>
  {/if}
</div>

<ConfirmDialog
  bind:open
  {title}
  {message}
  {confirmLabel}
  onconfirm={() => { lastAction = 'confirmed'; open = false }}
/>
