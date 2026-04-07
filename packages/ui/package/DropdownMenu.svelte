<script lang="ts">
  import { DropdownMenu } from 'bits-ui'
  import type { Snippet } from 'svelte'

  let {
    trigger,
    children,
  }: {
    trigger: Snippet<[Record<string, unknown>]>
    children?: Snippet
  } = $props()
</script>

<DropdownMenu.Root>
  <DropdownMenu.Trigger>
    {#snippet child({ props })}
      {@render trigger(props)}
    {/snippet}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content
    class="bg-bg border border-border rounded shadow-lg py-1 z-50 min-w-32"
  >
    {#if children}{@render children()}{/if}
  </DropdownMenu.Content>
</DropdownMenu.Root>
