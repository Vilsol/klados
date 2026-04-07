<script lang="ts">
  import DropdownMenu from '../DropdownMenu.svelte'
  import { DropdownMenu as BitsDropdown } from 'bits-ui'
</script>

<DropdownMenu>
  {#snippet trigger(props)}
    <button {...props}>Open menu</button>
  {/snippet}
  {#snippet children()}
    <BitsDropdown.Item class="px-3 py-1.5 text-sm text-fg hover:bg-surface-hover cursor-pointer">
      Item 1
    </BitsDropdown.Item>
  {/snippet}
</DropdownMenu>
