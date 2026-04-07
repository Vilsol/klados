import { notificationStore } from '$lib/stores/notification.svelte'

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
    notificationStore.push(e?.message ?? errorFallback, 'error')
  } finally {
    setBusy(false)
  }
}
