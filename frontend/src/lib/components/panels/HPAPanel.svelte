<script lang="ts">
  import {push} from "svelte-spa-router";
  import {SectionHeader, StatusBadge, DataTable} from "@klados/ui";

  let {obj, ctxName}: {obj: Record<string, any>; ctxName: string} = $props();

  const scaleTargetRef = $derived(obj.spec?.scaleTargetRef ?? {});
  const minReplicas = $derived<number>(obj.spec?.minReplicas ?? 1);
  const maxReplicas = $derived<number>(obj.spec?.maxReplicas ?? 1);
  const currentReplicas = $derived<number>(obj.status?.currentReplicas ?? 0);
  const desiredReplicas = $derived<number>(obj.status?.desiredReplicas ?? 0);
  const specMetrics = $derived<any[]>(obj.spec?.metrics ?? []);
  const currentMetrics = $derived<any[]>(obj.status?.currentMetrics ?? []);
  const behavior = $derived(obj.spec?.behavior);
  const conditions = $derived<any[]>(obj.status?.conditions ?? []);

  const kindPluralMap: Record<string, string> = {
    Deployment: "deployments",
    StatefulSet: "statefulsets",
    ReplicaSet: "replicasets",
    DaemonSet: "daemonsets",
  };

  function scaleTargetURL(): string {
    const apiVersion: string = scaleTargetRef.apiVersion ?? "";
    const kind: string = scaleTargetRef.kind ?? "";
    const slashIdx = apiVersion.indexOf("/");
    const group = slashIdx >= 0 ? apiVersion.slice(0, slashIdx) : "core";
    const version = slashIdx >= 0 ? apiVersion.slice(slashIdx + 1) : apiVersion;
    const plural = kindPluralMap[kind] ?? `${kind.toLowerCase()}s`;
    const ns = obj.metadata?.namespace ?? "_";
    return `/c/${ctxName}/${group}.${version}.${plural}/${ns}/${scaleTargetRef.name}`;
  }

  const gaugePercent = $derived(() => {
    const range = maxReplicas - minReplicas;
    if (range <= 0) {
      return 100;
    }
    return Math.min(100, Math.max(0, ((currentReplicas - minReplicas) / range) * 100));
  });

  const desiredPercent = $derived(() => {
    const range = maxReplicas - minReplicas;
    if (range <= 0) {
      return 100;
    }
    return Math.min(100, Math.max(0, ((desiredReplicas - minReplicas) / range) * 100));
  });

  function getMetricName(m: any): string {
    const type: string = m.type ?? "";
    if (type === "Resource") {
      return m.resource?.name ?? "—";
    }
    if (type === "Pods") {
      return m.pods?.metric?.name ?? "—";
    }
    if (type === "Object") {
      return m.object?.metric?.name ?? "—";
    }
    if (type === "External") {
      return m.external?.metric?.name ?? "—";
    }
    return "—";
  }

  function getMetricTarget(m: any): string {
    const type: string = m.type ?? "";
    if (type === "Resource") {
      const t = m.resource?.target;
      if (!t) {
        return "—";
      }
      if (t.type === "Utilization") {
        return `${t.averageUtilization ?? "—"}%`;
      }
      if (t.type === "AverageValue") {
        return t.averageValue ?? "—";
      }
      return t.value ?? "—";
    }
    if (type === "Pods") {
      const t = m.pods?.target;
      return t?.averageValue ?? "—";
    }
    if (type === "Object") {
      const t = m.object?.target;
      if (!t) {
        return "—";
      }
      if (t.type === "Value") {
        return t.value ?? "—";
      }
      return t.averageValue ?? "—";
    }
    if (type === "External") {
      const t = m.external?.target;
      if (!t) {
        return "—";
      }
      if (t.type === "Value") {
        return t.value ?? "—";
      }
      return t.averageValue ?? "—";
    }
    return "—";
  }

  function getCurrentMetric(m: any): string {
    if (currentMetrics.length === 0) {
      return "<unknown>";
    }
    const type: string = m.type ?? "";
    const name = getMetricName(m);
    const found = currentMetrics.find((c: any) => {
      if (c.type !== type) {
        return false;
      }
      return getMetricName(c) === name;
    });
    if (!found) {
      return "<unknown>";
    }
    if (type === "Resource") {
      return found.resource?.current?.averageUtilization != null
        ? `${found.resource.current.averageUtilization}%`
        : (found.resource?.current?.averageValue ?? "<unknown>");
    }
    if (type === "Pods") {
      return found.pods?.current?.averageValue ?? "<unknown>";
    }
    if (type === "Object") {
      return found.object?.current?.value ?? found.object?.current?.averageValue ?? "<unknown>";
    }
    if (type === "External") {
      return found.external?.current?.value ?? found.external?.current?.averageValue ?? "<unknown>";
    }
    return "<unknown>";
  }
</script>

<div class="p-4 space-y-6">
  <!-- Scale target -->
  {#if scaleTargetRef.name}
    <section>
      <SectionHeader>Scale Target</SectionHeader>
      <div class="flex items-center gap-2 text-xs">
        <span class="px-1.5 py-0.5 rounded bg-surface border border-border text-muted font-medium"> {scaleTargetRef.kind ?? ''} </span>
        <button onclick={() => push(scaleTargetURL())} class="text-accent hover:underline font-medium">{scaleTargetRef.name}</button>
      </div>
    </section>
  {/if}

  <!-- Replica gauge -->
  <section>
    <SectionHeader>Replicas</SectionHeader>
    <div class="space-y-2">
      <div class="relative h-6 bg-surface border border-border rounded-full overflow-hidden">
        <div class="absolute left-0 top-0 h-full bg-accent/30 rounded-full transition-all" style="width: {gaugePercent()}%"></div>
        {#if desiredReplicas !== currentReplicas}
          <div class="absolute top-0 h-full w-0.5 bg-yellow-400" style="left: {desiredPercent()}%"></div>
        {/if}
      </div>
      <div class="flex justify-between text-xs text-muted">
        <span>min: {minReplicas}</span>
        <span class="font-semibold text-fg"
          >current: {currentReplicas}{desiredReplicas !== currentReplicas ? ` (desired: ${desiredReplicas})` : ''}</span
        >
        <span>max: {maxReplicas}</span>
      </div>
    </div>
  </section>

  <!-- Metrics -->
  {#if specMetrics.length > 0}
    <section>
      <SectionHeader>Metrics</SectionHeader>
      <DataTable columns={[{ label: 'Type' }, { label: 'Name' }, { label: 'Target' }, { label: 'Current' }]} items={specMetrics}>
        {#snippet row(m)}
          <td class="px-2 py-1.5">
            <span class="px-1.5 py-0.5 rounded text-xs font-medium bg-surface border border-border">{m.type ?? '—'}</span>
          </td>
          <td class="px-2 py-1.5 font-mono">{getMetricName(m)}</td>
          <td class="px-2 py-1.5 font-mono">{getMetricTarget(m)}</td>
          <td class="px-2 py-1.5 font-mono text-muted">{getCurrentMetric(m)}</td>
        {/snippet}
      </DataTable>
    </section>
  {/if}

  <!-- Scaling behavior -->
  {#if behavior}
    <section>
      <SectionHeader>Scaling Behavior</SectionHeader>
      <div class="grid grid-cols-2 gap-4">
        {#each [{ label: 'Scale Up', key: 'scaleUp' }, { label: 'Scale Down', key: 'scaleDown' }] as dir}
          {#if behavior[dir.key]}
            {@const d = behavior[dir.key]}
            <div class="bg-surface border border-border rounded-lg p-3 space-y-2 text-xs">
              <div class="font-medium text-muted">{dir.label}</div>
              {#if d.stabilizationWindowSeconds != null}
                <div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
                  <span class="text-muted">Stabilization</span>
                  <span class="font-mono">{d.stabilizationWindowSeconds}s</span>
                  {#if d.selectPolicy}
                    <span class="text-muted">Select Policy</span>
                    <span class="font-mono">{d.selectPolicy}</span>
                  {/if}
                </div>
              {/if}
              {#if d.policies?.length}
                <div class="space-y-1">
                  {#each d.policies as policy}
                    <div class="flex items-center gap-2 text-muted">
                      <span class="font-mono">{policy.value} {policy.type}</span>
                      <span>per {policy.periodSeconds}s</span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        {/each}
      </div>
    </section>
  {/if}

  <!-- Conditions -->
  {#if conditions.length > 0}
    <section>
      <SectionHeader>Conditions</SectionHeader>
      <DataTable columns={[{ label: 'Type' }, { label: 'Status' }, { label: 'Reason' }, { label: 'Message' }]} items={conditions}>
        {#snippet row(cond)}
          <td class="px-2 py-1.5 font-mono">{cond.type}</td>
          <td class="px-2 py-1.5"><StatusBadge status={cond.status}>{cond.status}</StatusBadge></td>
          <td class="px-2 py-1.5 text-muted">{cond.reason ?? '—'}</td>
          <td class="px-2 py-1.5 text-muted max-w-xs truncate">{cond.message ?? '—'}</td>
        {/snippet}
      </DataTable>
    </section>
  {/if}
</div>
