<script lang="ts">
  import { Dialog } from 'bits-ui'
  import * as ClusterService from '../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice.js'
  import * as AppService from '../../../bindings/github.com/Vilsol/klados/internal/services/appservice.js'

  let {
    open = $bindable(false),
    onsuccess,
  }: {
    open: boolean
    onsuccess: (contexts: any[]) => void
  } = $props()

  let mode = $state<'path' | 'paste'>('path')
  let filePath = $state('')
  let yamlContent = $state('')
  let error = $state('')
  let loading = $state(false)

  async function browse() {
    try {
      const path = await AppService.BrowseKubeconfigFile()
      if (path) filePath = path
    } catch (e: any) {
      error = e?.message ?? String(e)
    }
  }

  async function submit() {
    error = ''
    loading = true
    try {
      let contexts: any[]
      if (mode === 'path') {
        if (!filePath.trim()) { error = 'Enter a file path'; return }
        contexts = await ClusterService.AddKubeconfigPath(filePath.trim())
      } else {
        if (!yamlContent.trim()) { error = 'Paste kubeconfig YAML'; return }
        contexts = await ClusterService.ImportKubeconfigContent(yamlContent.trim())
      }
      open = false
      filePath = ''
      yamlContent = ''
      onsuccess(contexts ?? [])
    } catch (e: any) {
      error = e?.message ?? String(e)
    } finally {
      loading = false
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[480px] max-w-[90vw]"
    >
      <Dialog.Title class="text-base font-semibold mb-4">Import Kubeconfig</Dialog.Title>

      <!-- Mode tabs -->
      <div class="flex gap-1 mb-4 border-b border-border">
        <button
          onclick={() => { mode = 'path'; error = '' }}
          class="px-3 py-1.5 text-xs font-medium border-b-2 transition-colors
            {mode === 'path' ? 'border-accent text-accent' : 'border-transparent text-muted hover:text-fg'}"
        >
          File Path
        </button>
        <button
          onclick={() => { mode = 'paste'; error = '' }}
          class="px-3 py-1.5 text-xs font-medium border-b-2 transition-colors
            {mode === 'paste' ? 'border-accent text-accent' : 'border-transparent text-muted hover:text-fg'}"
        >
          Paste YAML
        </button>
      </div>

      {#if mode === 'path'}
        <div class="flex gap-2 mb-4">
          <input
            type="text"
            bind:value={filePath}
            placeholder="/path/to/kubeconfig.yaml"
            class="flex-1 px-3 py-1.5 text-sm rounded border border-border bg-surface focus:outline-none focus:border-accent"
          />
          <button
            onclick={browse}
            class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors shrink-0"
          >
            Browse
          </button>
        </div>
      {:else}
        <textarea
          bind:value={yamlContent}
          placeholder="Paste kubeconfig YAML here..."
          rows={10}
          class="w-full px-3 py-2 text-xs font-mono rounded border border-border bg-surface focus:outline-none focus:border-accent resize-none mb-4"
        ></textarea>
      {/if}

      {#if error}
        <p class="text-xs text-destructive mb-3">{error}</p>
      {/if}

      <div class="flex justify-end gap-2">
        <Dialog.Close
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
        >
          Cancel
        </Dialog.Close>
        <button
          onclick={submit}
          disabled={loading}
          class="px-3 py-1.5 text-sm rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          {loading ? 'Importing...' : 'Import'}
        </button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
