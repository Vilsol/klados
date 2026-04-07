<script lang="ts">
  import { Button } from '@klados/ui'

  let {
    variant = 'primary',
    disabled = false,
    label = 'Click me',
  }: {
    variant?: 'primary' | 'destructive' | 'ghost' | 'outline'
    disabled?: boolean
    label?: string
  } = $props()
</script>

<Button {variant} {disabled}>{label}</Button>
