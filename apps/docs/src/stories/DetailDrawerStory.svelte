<script lang="ts">
  import { DetailDrawer, CodeBlock, Button } from '@klados/ui'

  let {
    resourceName = 'nginx-abc123',
    resourceNamespace = 'default',
    gvr = 'core.v1.pods',
  }: {
    resourceName?: string
    resourceNamespace?: string
    gvr?: string
  } = $props()

  const item = $derived({
    apiVersion: 'v1',
    kind: 'Pod',
    metadata: { name: resourceName, namespace: resourceNamespace },
    status: { phase: 'Running' },
  })

  let show = $state(true)
</script>

<div class="relative h-96 bg-bg border border-border rounded overflow-hidden">
  <div class="p-4">
    {#if !show}
      <Button onclick={() => (show = true)}>Open drawer</Button>
    {:else}
      <p class="text-sm text-muted">Click × to close the drawer</p>
    {/if}
  </div>

  {#if show}
    <DetailDrawer {item} ctxName="prod" {gvr} onclose={() => (show = false)}>
      {#snippet children({ obj })}
        <div class="p-4 overflow-auto h-full">
          <CodeBlock value={JSON.stringify(obj, null, 2)} lang="json" />
        </div>
      {/snippet}
    </DetailDrawer>
  {/if}
</div>
