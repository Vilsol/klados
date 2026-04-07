<script lang="ts">
  import { Dialog, Button } from '@klados/ui'

  let {
    title = 'Dialog title',
    description = 'Dialog description text.',
    showFooter = false,
  }: {
    title?: string
    description?: string
    showFooter?: boolean
  } = $props()

  let open = $state(false)
</script>

<div class="p-8">
  <Dialog bind:open {title} {description}>
    {#snippet trigger()}
      <Button onclick={() => (open = true)}>Open Dialog</Button>
    {/snippet}
    {#snippet children()}
      <p class="text-sm text-muted">Dialog body content goes here.</p>
      {#if showFooter}
        <div class="flex justify-end gap-2 mt-4">
          <Button variant="outline" onclick={() => (open = false)}>Cancel</Button>
          <Button onclick={() => (open = false)}>Confirm</Button>
        </div>
      {/if}
    {/snippet}
  </Dialog>
</div>
