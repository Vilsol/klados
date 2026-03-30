<script lang="ts">
  let { resource }: { resource: Record<string, any> } = $props()

  const count = $derived<number>(
    resource?.status?.taintCount ?? resource?.spec?.taints?.length ?? 0
  )
</script>

<span
  class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium border"
  class:text-destructive={count > 0}
  class:border-destructive={count > 0}
  class:bg-destructive={count > 0}
  class:text-muted={count === 0}
  class:border-border={count === 0}
>
  {count} {count === 1 ? 'taint' : 'taints'}
</span>
