<script lang="ts">
  import { VirtualLogViewer } from '@klados/ui'

  let {
    lineCount = 20,
    includeErrors = false,
    showTimestamps = false,
  }: {
    lineCount?: number
    includeErrors?: boolean
    showTimestamps?: boolean
  } = $props()

  const ts = (i: number) => `2024-01-15T${String(10 + Math.floor(i / 60)).padStart(2, '0')}:${String(i % 60).padStart(2, '0')}:00Z`

  const lines = $derived(
    Array.from({ length: lineCount }, (_, i) => {
      const prefix = showTimestamps ? `${ts(i)} ` : ''
      if (includeErrors && i === 5) return `${prefix}ERROR failed to connect to database: connection refused`
      if (includeErrors && i === 12) return `${prefix}WARN retry attempt 3/5`
      if (i % 4 === 0) return `${prefix}INFO server listening on :8080 version=1.2.3`
      if (i % 4 === 1) return `${prefix}INFO request completed method=GET path=/health status=200 duration=2ms`
      if (i % 4 === 2) return `${prefix}{"level":"info","msg":"processed event","id":"evt-${i}","ts":"${new Date().toISOString()}"}`
      return `${prefix}DEBUG cache hit key=user:${i} ttl=300s`
    }),
  )
</script>

<div class="h-96 border border-border rounded overflow-hidden">
  <VirtualLogViewer {lines} eofReached={true} {showTimestamps} />
</div>
