<script lang="ts">
  import { Dialog } from 'bits-ui'

  let {
    open = $bindable(false),
    title = 'Confirm',
    message = 'Are you sure?',
    confirmLabel = 'Confirm',
    onconfirm,
  }: {
    open: boolean
    title?: string
    message?: string
    confirmLabel?: string
    onconfirm: () => void
  } = $props()
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-96 max-w-[90vw]"
    >
      <Dialog.Title class="text-base font-semibold mb-2">{title}</Dialog.Title>
      <Dialog.Description class="text-sm text-muted mb-6">{message}</Dialog.Description>

      <div class="flex justify-end gap-2">
        <Dialog.Close
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
        >
          Cancel
        </Dialog.Close>
        <button
          onclick={() => { open = false; onconfirm() }}
          class="px-3 py-1.5 text-sm rounded bg-destructive text-destructive-fg hover:opacity-90 transition-opacity"
        >
          {confirmLabel}
        </button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
