<script lang="ts">
  import { Icon } from '@klados/ui'
  import { Server, AlertCircle, CheckCircle, Settings, Trash2, Plus, RefreshCw } from 'lucide-svelte'

  const iconMap: Record<string, any> = {
    Server, AlertCircle, CheckCircle, Settings, Trash2, Plus, RefreshCw,
  }

  let {
    iconName = 'Server',
    size = 16,
  }: {
    iconName?: string
    size?: number
  } = $props()

  const icon = $derived(iconMap[iconName] ?? Server)
</script>

<Icon {icon} {size} />
