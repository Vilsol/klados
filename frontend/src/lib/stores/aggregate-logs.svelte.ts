const COLORS = [
  'var(--log-color-1)',
  'var(--log-color-2)',
  'var(--log-color-3)',
  'var(--log-color-4)',
  'var(--log-color-5)',
  'var(--log-color-6)',
  'var(--log-color-7)',
  'var(--log-color-8)',
]

const MAX_LINES = 10_000

export interface AggregateEntry {
  pod: string
  text: string
  color: string
  timestamp: number
}

interface StreamEntry {
  streamId: string
  ws: WebSocket
  color: string
}

export class AggregateLogStore {
  streams = $state<Map<string, StreamEntry>>(new Map())
  lines = $state<AggregateEntry[]>([])
  showPodPrefix = $state(true)

  private colorIdx = 0

  addStream(pod: string, streamId: string, ws: WebSocket) {
    const color = COLORS[this.colorIdx % COLORS.length]
    this.colorIdx++
    this.streams.set(pod, { streamId, ws, color })
  }

  appendLine(pod: string, text: string) {
    const color = this.streams.get(pod)?.color ?? COLORS[0]
    if (this.lines.length >= MAX_LINES) this.lines.shift()
    this.lines.push({ pod, text, color, timestamp: Date.now() })
  }

  markEnded(pod: string) {
    this.appendLine(pod, '[stream ended]')
    this.streams.delete(pod)
  }

  destroy() {
    for (const { ws } of this.streams.values()) ws.close()
    this.streams.clear()
    this.lines = []
    this.colorIdx = 0
  }
}
