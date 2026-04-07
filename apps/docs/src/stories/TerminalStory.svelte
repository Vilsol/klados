<script lang="ts">
  import { Terminal } from '@klados/ui'

  let {
    sessionID = 'story-session-1',
  }: {
    sessionID?: string
  } = $props()
</script>

<div class="h-64 border border-border rounded overflow-hidden">
  <Terminal
    {sessionID}
    streamingConfig={{ port: 9999, token: 'story-token' }}
  />
</div>
