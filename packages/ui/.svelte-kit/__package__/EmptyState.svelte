<script lang="ts">
  let {
    message = 'None',
    size = 'xs',
  }: {
    message?: string
    size?: 'xs' | 'sm'
  } = $props()
</script>

<p class="text-muted {size === 'sm' ? 'text-sm' : 'text-xs'}">{message}</p>
