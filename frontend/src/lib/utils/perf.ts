const timers = new Map<string, number>()

export function perfStart(label: string): void {
  timers.set(label, performance.now())
  console.debug(`[perf] ${label} — started`)
}

export function perfActive(label: string): boolean {
  return timers.has(label)
}

export function perfMark(label: string, message: string): void {
  const start = timers.get(label)
  if (start == null) return
  console.debug(`[perf] ${label} — ${message} (+${(performance.now() - start).toFixed(1)}ms)`)
}

export function perfEnd(label: string, message?: string): void {
  const start = timers.get(label)
  const elapsed = start != null ? `${(performance.now() - start).toFixed(1)}ms` : 'unknown'
  console.debug(`[perf] ${label} — ${message ?? 'done'} (total: ${elapsed})`)
  timers.delete(label)
}
