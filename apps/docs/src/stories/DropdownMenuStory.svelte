<script lang="ts">
  import { DropdownMenu, Button } from '@klados/ui'

  let {
    withIcons = false,
  }: {
    withIcons?: boolean
  } = $props()
</script>

<div class="flex items-center justify-center p-8">
  <DropdownMenu>
    {#snippet trigger(props)}
      <Button variant="outline" {...props}>Actions ▾</Button>
    {/snippet}
    {#snippet children()}
      <div class="px-3 py-1.5 text-sm text-fg hover:bg-surface-hover cursor-pointer">
        {#if withIcons}⚙ {/if}Edit
      </div>
      <div class="px-3 py-1.5 text-sm text-fg hover:bg-surface-hover cursor-pointer">
        {#if withIcons}📋 {/if}Duplicate
      </div>
      <div class="px-3 py-1.5 text-sm text-destructive hover:bg-surface-hover cursor-pointer">
        {#if withIcons}🗑 {/if}Delete
      </div>
    {/snippet}
  </DropdownMenu>
</div>
