<script lang="ts">
  import { LogViewer } from '@klados/ui'

  let {
    streamID = 'story-stream-1',
  }: {
    streamID?: string
  } = $props()
</script>

<div class="h-64 border border-border rounded overflow-hidden">
  <LogViewer
    {streamID}
    streamingConfig={{ port: 9999, token: 'story-token' }}
  />
</div>
