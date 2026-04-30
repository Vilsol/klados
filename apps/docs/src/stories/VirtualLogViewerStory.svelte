<script lang="ts">
  import { VirtualLogViewer } from '@klados/ui'

  let {
    lineCount = 20,
    includeErrors = false,
    showTimestamps = false,
    longLines = false,
    manyMatches = false,
  }: {
    lineCount?: number
    includeErrors?: boolean
    showTimestamps?: boolean
    longLines?: boolean
    manyMatches?: boolean
  } = $props()

  const ts = (i: number) => `2024-01-15T${String(10 + Math.floor(i / 60)).padStart(2, '0')}:${String(i % 60).padStart(2, '0')}:00Z`

  const lines = $derived(
    Array.from({ length: lineCount }, (_, i) => {
      const prefix = showTimestamps ? `${ts(i)} ` : ''
      const tail = longLines ? ' ' + 'x'.repeat(400) : ''
      const targetWord = manyMatches ? ' target' : ''
      if (includeErrors && i === 5) return `${prefix}ERROR failed to connect to database: connection refused${targetWord}${tail}`
      if (includeErrors && i === 12) return `${prefix}WARN retry attempt 3/5${targetWord}${tail}`
      if (i % 4 === 0) return `${prefix}INFO server listening on :8080 version=1.2.3${targetWord}${tail}`
      if (i % 4 === 1) return `${prefix}INFO request completed method=GET path=/health status=200 duration=2ms${targetWord}${tail}`
      if (i % 4 === 2) return `${prefix}{"level":"info","msg":"processed event","id":"evt-${i}","ts":"${new Date().toISOString()}"${manyMatches ? ',"tag":"target"' : ''}}${tail}`
      return `${prefix}DEBUG cache hit key=user:${i} ttl=300s${targetWord}${tail}`
    }),
  )
</script>

<div class="h-96 border border-border rounded overflow-hidden">
  <VirtualLogViewer {lines} eofReached={true} {showTimestamps} />
</div>
