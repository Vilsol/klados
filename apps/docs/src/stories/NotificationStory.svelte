<script lang="ts">
  import { onMount } from 'svelte'
  import { Notification, notificationStore } from '@klados/ui'

  let {
    type = 'info',
    message = 'Sample notification message',
    withDetails = false,
  }: {
    type?: 'info' | 'success' | 'error'
    message?: string
    withDetails?: boolean
  } = $props()

  onMount(() => {
    notificationStore.push(message, type, withDetails ? { details: 'Error details: connection refused' } : undefined)
  })
</script>

<div class="relative h-32 w-full">
  <Notification />
</div>
