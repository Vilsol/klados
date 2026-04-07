<script lang="ts">
  import Dialog from '../Dialog.svelte'

  let open = $state(false)
</script>

<Dialog bind:open title="Test Dialog" description="Test description">
  {#snippet trigger()}
    <button onclick={() => (open = true)}>Open</button>
  {/snippet}
  {#snippet children()}
    <p>Dialog content</p>
  {/snippet}
</Dialog>
