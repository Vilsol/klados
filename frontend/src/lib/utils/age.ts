export function formatAge(timestamp: string, now = Date.now()): string {
  if (!timestamp) return ''
  const created = new Date(timestamp)
  if (isNaN(created.getTime())) return timestamp

  const seconds = Math.floor((now - created.getTime()) / 1000)
  if (seconds < 0) return '0s'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`
  return `${Math.floor(seconds / 86400)}d`
}
