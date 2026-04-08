<script lang="ts">
  import { SectionHeader, StatusBadge } from '@klados/ui'

  let { obj, ctxName }: { obj: Record<string, any>; ctxName: string } = $props()

  const endpoints = $derived<any[]>(obj.endpoints ?? [])
  const ports = $derived<any[]>(obj.ports ?? [])
  const addressType = $derived<string>(obj.addressType ?? '—')
  const ns = $derived<string>(obj.metadata?.namespace ?? '')
</script>

<div class="flex flex-col gap-6 p-4 overflow-auto">
  <section>
    <SectionHeader>
      Address Type: <span class="ml-1 normal-case tracking-normal font-mono">{addressType}</span>
    </SectionHeader>
  </section>

  {#if ports.length > 0}
    <section>
      <SectionHeader>Ports</SectionHeader>
      <table class="w-full text-xs">
        <thead class="bg-surface">
          <tr>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Name</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Port</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Protocol</th>
          </tr>
        </thead>
        <tbody>
          {#each ports as port}
            <tr class="border-t border-border">
              <td class="px-2 py-1.5 font-mono">{port.name ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{port.port ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{port.protocol ?? '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}

  {#if endpoints.length > 0}
    <section>
      <SectionHeader>Addresses</SectionHeader>
      <table class="w-full text-xs">
        <thead class="bg-surface">
          <tr>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Address</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Node</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Ready</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Serving</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Terminating</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Target Ref</th>
          </tr>
        </thead>
        <tbody>
          {#each endpoints as ep}
            {@const cond = ep.conditions ?? {}}
            {@const ref = ep.targetRef}
            <tr class="border-t border-border">
              <td class="px-2 py-1.5 font-mono">{ep.addresses?.[0] ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{ep.nodeName ?? '—'}</td>
              <td class="px-2 py-1.5">
                <StatusBadge status={cond.ready ?? false}>{cond.ready ? 'True' : 'False'}</StatusBadge>
              </td>
              <td class="px-2 py-1.5">
                <StatusBadge status={cond.serving ?? false}>{cond.serving ? 'True' : 'False'}</StatusBadge>
              </td>
              <td class="px-2 py-1.5">
                {#if cond.terminating}
                  <StatusBadge status={false}>True</StatusBadge>
                {:else}
                  <StatusBadge status="Unknown">False</StatusBadge>
                {/if}
              </td>
              <td class="px-2 py-1.5">
                {#if ref?.kind === 'Pod'}
                  <a
                    href="/c/{ctxName}/core.v1.pods/{ref.namespace ?? ns}/{ref.name}"
                    class="text-accent hover:underline font-mono"
                  >{ref.name}</a>
                {:else if ref}
                  <span class="font-mono text-muted">{ref.kind}/{ref.name}</span>
                {:else}
                  <span class="text-muted">—</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}
</div>
