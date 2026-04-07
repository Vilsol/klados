import { notificationStore } from '$lib/stores/notification.svelte'

export function unwrapError(e: any, fallback = 'Operation failed'): string {
  const raw: string = typeof e === 'string' ? e : (e?.message ?? String(e ?? fallback))
  try {
    const parsed = JSON.parse(raw)
    if (typeof parsed?.message === 'string') {
      return typeof parsed?.kind === 'string'
        ? `${parsed.kind}: ${parsed.message}`
        : parsed.message
    }
  } catch {
    // not JSON
  }
  return raw || fallback
}

export async function withBusy(
  setBusy: (v: boolean) => void,
  fn: () => Promise<void>,
  successMsg: string,
  errorFallback = 'Operation failed',
  onSuccess?: () => void,
): Promise<void> {
  setBusy(true)
  try {
    await fn()
    notificationStore.push(successMsg, 'success')
    onSuccess?.()
  } catch (e: any) {
    notificationStore.push(unwrapError(e, errorFallback), 'error')
  } finally {
    setBusy(false)
  }
}
