<script lang="ts">
  import { Dialog } from 'bits-ui'
  import type { Snippet } from 'svelte'

  let {
    open = $bindable(false),
    title,
    description,
    trigger,
    children,
  }: {
    open?: boolean
    title?: string
    description?: string
    trigger?: Snippet
    children?: Snippet
  } = $props()
</script>

<Dialog.Root bind:open>
  {#if trigger}
    <Dialog.Trigger>
      {#snippet child({ props })}
        {@render trigger()}
      {/snippet}
    </Dialog.Trigger>
  {/if}
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[28rem] max-w-[90vw]"
    >
      {#if title}
        <Dialog.Title class="text-base font-semibold mb-1">{title}</Dialog.Title>
      {/if}
      {#if description}
        <Dialog.Description class="text-sm text-muted mb-4">{description}</Dialog.Description>
      {/if}
      {#if children}{@render children()}{/if}
      <Dialog.Close
        class="absolute top-4 right-4 text-muted hover:text-fg transition-colors text-lg leading-none"
        aria-label="Close"
      >
        ×
      </Dialog.Close>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
