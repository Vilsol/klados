<script lang="ts">
  let { resource, onclose }: { resource: Record<string, any>; onclose?: () => void } = $props()

  function copyName() {
    navigator.clipboard.writeText(resource?.metadata?.name ?? '')
    onclose?.()
  }
</script>

<button
  class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover transition-colors"
  onclick={copyName}
>
  Copy node name
</button>
