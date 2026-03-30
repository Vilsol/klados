<script lang="ts">
  import { onMount } from 'svelte'
  import * as ResourceService from '../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import PortForwardDialog from '$lib/components/PortForwardDialog.svelte'

  let { obj, ctxName }: { obj: Record<string, any>; ctxName: string } = $props()

  let pfPort = $state<number | null>(null)

  const selector = $derived<Record<string, string>>(obj.spec?.selector ?? {})
  const ports = $derived<any[]>(obj.spec?.ports ?? [])

  let endpoints = $state<any | null>(null)
  let endpointError = $state('')

  const ns = $derived<string>(obj.metadata?.namespace ?? '')
  const svcName = $derived<string>(obj.metadata?.name ?? '')

  $effect(() => {
    if (!ctxName || !ns || !svcName) return
    endpoints = null
    endpointError = ''
    ResourceService.GetResource(ctxName, 'core.v1.endpoints', ns, svcName)
      .then((r: any) => { endpoints = r })
      .catch((e: any) => { endpointError = String(e) })
  })

  const subsets = $derived<any[]>(endpoints?.subsets ?? [])

  interface BackingPod { name: string; ip: string }

  const backingPods = $derived<BackingPod[]>(
    subsets.flatMap((s: any) =>
      (s.addresses ?? []).map((a: any) => ({
        name: a.targetRef?.name ?? a.ip,
        ip: a.ip,
      }))
    )
  )
</script>

<div class="flex flex-col gap-4 p-4 overflow-auto h-full">
  <!-- Selector -->
  <section>
    <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Selector</h3>
    {#if Object.keys(selector).length > 0}
      <div class="flex flex-wrap gap-1.5">
        {#each Object.entries(selector) as [k, v]}
          <span class="text-xs bg-surface-hover border border-border rounded px-2 py-0.5 font-mono">
            {k}={v}
          </span>
        {/each}
      </div>
    {:else}
      <p class="text-sm text-muted">No selector</p>
    {/if}
  </section>

  <!-- Ports -->
  {#if ports.length > 0}
    <section>
      <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Ports</h3>
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-muted text-xs border-b border-border">
            <th class="pb-1 pr-4 font-medium">Name</th>
            <th class="pb-1 pr-4 font-medium">Port</th>
            <th class="pb-1 pr-4 font-medium">Protocol</th>
            <th class="pb-1 pr-4 font-medium">Target</th>
            <th class="pb-1 font-medium"></th>
          </tr>
        </thead>
        <tbody>
          {#each ports as p}
            <tr class="border-b border-border/50">
              <td class="py-1.5 pr-4 text-muted">{p.name ?? '—'}</td>
              <td class="py-1.5 pr-4 font-mono">{p.port}</td>
              <td class="py-1.5 pr-4">{p.protocol ?? 'TCP'}</td>
              <td class="py-1.5 pr-4 font-mono">{p.targetPort ?? '—'}</td>
              <td class="py-1.5">
                <button
                  onclick={() => pfPort = p.port}
                  class="text-xs font-mono text-accent hover:underline"
                  title="Forward port {p.port}"
                  aria-label="Forward port {p.port}"
                >↗</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}

  <!-- Endpoints / backing pods -->
  <section>
    <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Endpoints</h3>
    {#if endpointError}
      <p class="text-sm text-muted">{endpointError}</p>
    {:else if endpoints === null}
      <p class="text-sm text-muted">Loading…</p>
    {:else if backingPods.length === 0}
      <p class="text-sm text-muted">No endpoints</p>
    {:else}
      <div class="flex flex-col gap-1">
        {#each backingPods as pod}
          <div class="flex items-center gap-2 text-sm">
            <span class="font-medium">{pod.name}</span>
            <span class="text-muted font-mono text-xs">{pod.ip}</span>
          </div>
        {/each}
      </div>
    {/if}
  </section>
</div>

{#if pfPort !== null}
  <PortForwardDialog
    prefillContext={ctxName}
    prefillNamespace={ns}
    prefillTargetKind="selector"
    prefillTarget={svcName}
    prefillGVR="core.v1.services"
    prefillRemotePort={pfPort}
    onclose={() => pfPort = null}
  />
{/if}
