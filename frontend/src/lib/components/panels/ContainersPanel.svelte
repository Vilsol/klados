<script lang="ts">
  import PortForwardDialog from "$lib/components/PortForwardDialog.svelte";
  import PortButton from "$lib/components/PortButton.svelte";
  import {SectionHeader, StatusBadge, DataTable} from "@klados/ui";
  import {toggleSet} from "$lib/utils/collections";
  import type {KubernetesResource} from "$lib/types";

  let {obj, ctxName = ""}: {obj: Record<string, KubernetesResource>; ctxName?: string} = $props();

  let pfPort = $state<number | null>(null);

  const containers = $derived<KubernetesResource[]>(obj.spec?.containers ?? []);
  const initContainers = $derived<KubernetesResource[]>(obj.spec?.initContainers ?? []);
  const conditions = $derived<KubernetesResource[]>(obj.status?.conditions ?? []);

  function containerStatus(name: string): KubernetesResource {
    const statuses: KubernetesResource[] = obj.status?.containerStatuses ?? [];
    return statuses.find((s: KubernetesResource) => s.name === name);
  }

  function initContainerStatus(name: string): KubernetesResource {
    const statuses: KubernetesResource[] = obj.status?.initContainerStatuses ?? [];
    return statuses.find((s: KubernetesResource) => s.name === name);
  }

  function stateLabel(status: KubernetesResource): string {
    if (!status) {
      return "Unknown";
    }
    if (status.state?.running) {
      return "Running";
    }
    if (status.state?.waiting) {
      return `Waiting: ${status.state.waiting.reason ?? ""}`;
    }
    if (status.state?.terminated) {
      return `Terminated: ${status.state.terminated.reason ?? ""}`;
    }
    return "Unknown";
  }

  let expandedEnv = $state<Set<string>>(new Set());

  let showInitContainers = $state(false);
</script>

<div class="flex flex-col gap-4 p-4 overflow-auto">
  <!-- Main containers -->
  <section>
    <SectionHeader>Containers</SectionHeader>
    <div class="flex flex-col gap-3">
      {#each containers as c}
        {@const status = containerStatus(c.name)}
        <div class="bg-surface border border-border rounded-lg p-3">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-medium">{c.name}</span>
            <StatusBadge status={!!status?.ready} mode="pill">{stateLabel(status)}</StatusBadge>
          </div>
          <p class="text-xs font-mono text-muted truncate mb-2">{c.image}</p>

          <div class="grid grid-cols-3 gap-2 text-xs mb-2">
            {#if c.resources?.requests?.cpu || c.resources?.limits?.cpu}
              <div class="text-muted">CPU</div>
              <div class="font-mono">{c.resources?.requests?.cpu ?? '—'}</div>
              <div class="font-mono">{c.resources?.limits?.cpu ?? '—'}</div>
            {/if}
            {#if c.resources?.requests?.memory || c.resources?.limits?.memory}
              <div class="text-muted">Memory</div>
              <div class="font-mono">{c.resources?.requests?.memory ?? '—'}</div>
              <div class="font-mono">{c.resources?.limits?.memory ?? '—'}</div>
            {/if}
          </div>

          {#if status?.restartCount > 0}
            <p class="text-xs text-muted">Restarts: <span class="text-fg">{status.restartCount}</span></p>
          {/if}

          {#if c.ports?.length}
            <div class="flex flex-wrap gap-1 mt-1">
              {#each c.ports as p}
                <PortButton port={p.containerPort} protocol={p.protocol ?? 'TCP'} onclick={() => pfPort = p.containerPort} />
              {/each}
            </div>
          {/if}

          {#if c.env?.length}
            <button
              type="button"
              onclick={() => expandedEnv = toggleSet(expandedEnv, c.name)}
              class="text-xs text-accent hover:underline mt-1"
            >
              {expandedEnv.has(c.name) ? '▾' : '▸'} {c.env.length} env var{c.env.length !== 1 ? 's' : ''}
            </button>
            {#if expandedEnv.has(c.name)}
              <div class="mt-1.5 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 pl-3">
                {#each c.env as e}
                  <span class="text-xs font-mono text-accent">{e.name}</span>
                  <span class="text-xs font-mono text-muted truncate"> {e.value ?? (e.valueFrom ? '(from secret/configmap)' : '—')} </span>
                {/each}
              </div>
            {/if}
          {/if}

          {#if c.volumeMounts?.length}
            <div class="mt-1.5 text-xs text-muted">Mounts: {c.volumeMounts.map((m: KubernetesResource) => m.mountPath).join(', ')}</div>
          {/if}
        </div>
      {/each}
    </div>
  </section>

  <!-- Init containers (collapsible) -->
  {#if initContainers.length > 0}
    <section>
      <button
        type="button"
        onclick={() => showInitContainers = !showInitContainers}
        class="text-xs font-semibold text-muted uppercase tracking-wide mb-2 flex items-center gap-1"
      >
        {showInitContainers ? '▾' : '▸'}
        Init Containers ({initContainers.length})
      </button>
      {#if showInitContainers}
        <div class="flex flex-col gap-2">
          {#each initContainers as c}
            {@const status = initContainerStatus(c.name)}
            <div class="bg-surface border border-border rounded-lg p-3">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium">{c.name}</span>
                <span class="text-xs px-2 py-0.5 rounded-full bg-surface-hover text-muted">{stateLabel(status)}</span>
              </div>
              <p class="text-xs font-mono text-muted truncate mt-1">{c.image}</p>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  <!-- Pod conditions -->
  {#if conditions.length > 0}
    <section>
      <SectionHeader>Conditions</SectionHeader>
      <DataTable columns={[{ label: 'Type' }, { label: 'Status' }, { label: 'Reason' }]} items={conditions}>
        {#snippet row(cond)}
          <td class="px-2 py-1.5 font-mono">{cond.type}</td>
          <td class="px-2 py-1.5">
            <span class="{cond.status === 'True' ? 'text-green-600 dark:text-green-400' : 'text-muted'}">{cond.status}</span>
          </td>
          <td class="px-2 py-1.5 text-muted">{cond.reason ?? '—'}</td>
        {/snippet}
      </DataTable>
    </section>
  {/if}
</div>

{#if pfPort !== null}
  <PortForwardDialog
    prefillContext={ctxName}
    prefillNamespace={obj.metadata?.namespace ?? ''}
    prefillTargetKind="pod"
    prefillTarget={obj.metadata?.name ?? ''}
    prefillGVR=""
    prefillRemotePort={pfPort}
    onclose={() => pfPort = null}
  />
{/if}
