<script lang="ts">
  import { Badge } from '@klados/ui'

  let {
    variant = 'default',
    text = 'Badge',
  }: {
    variant?: 'default' | 'success' | 'warning' | 'destructive'
    text?: string
  } = $props()
</script>

<Badge {variant}>{text}</Badge>
