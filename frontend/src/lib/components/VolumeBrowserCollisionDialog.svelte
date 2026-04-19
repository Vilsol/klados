<script lang="ts">
  import {Dialog} from "bits-ui";

  interface Props {
    open: boolean;
    existingPodName: string;
    pvcName: string;
    onattach: () => void;
    onreplace: () => void;
    oncancel: () => void;
  }

  let {open = $bindable(), existingPodName, pvcName, onattach, onreplace, oncancel}: Props = $props();

  function pick(fn: () => void) {
    fn();
    open = false;
  }
</script>

<Dialog.Root bind:open onOpenChange={(v) => { if (!v) oncancel(); }}>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[460px] max-w-[92vw]"
    >
      <Dialog.Title class="text-base font-semibold mb-2">Browser pod already exists</Dialog.Title>
      <Dialog.Description class="text-sm text-muted mb-4">
        A browser pod <span class="text-fg font-mono text-xs">{existingPodName}</span> is already attached
        to <span class="text-fg font-mono text-xs">{pvcName}</span>. Attach to it, replace it, or cancel.
      </Dialog.Description>

      <div class="flex justify-end gap-2">
        <button
          type="button"
          onclick={() => pick(oncancel)}
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
        >Cancel</button>
        <button
          type="button"
          onclick={() => pick(onreplace)}
          class="px-3 py-1.5 text-sm rounded border border-destructive text-destructive hover:bg-destructive/10 transition-colors"
        >Replace</button>
        <button
          type="button"
          onclick={() => pick(onattach)}
          class="px-3 py-1.5 text-sm rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity"
        >Attach</button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
