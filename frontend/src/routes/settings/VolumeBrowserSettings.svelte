<script lang="ts">
  import {KeyValuePairEditor, Select} from "@klados/ui";
  import type {VolumeBrowserConfig} from "$lib/stores/preferences.svelte";

  interface Props {
    value: VolumeBrowserConfig;
    onchange?: (next: VolumeBrowserConfig) => void;
  }

  let {value = $bindable(), onchange}: Props = $props();

  // Tolerations textarea state (parsed on blur)
  let tolerationsText = $state<string>(
    value.tolerations ? JSON.stringify(value.tolerations, null, 2) : ""
  );
  let tolerationsError = $state<string>("");

  // Node selector: KeyValuePairEditor expects [string,string][]
  let nodeSelectorPairs = $state<[string, string][]>(
    value.nodeSelector ? Object.entries(value.nodeSelector) : []
  );

  // Resources toggle
  let resourcesEnabled = $state<boolean>(value.resources != null);
  let cpuRequest = $state<string>(value.resources?.requests?.cpu ?? "");
  let memoryRequest = $state<string>(value.resources?.requests?.memory ?? "");
  let cpuLimit = $state<string>(value.resources?.limits?.cpu ?? "");
  let memoryLimit = $state<string>(value.resources?.limits?.memory ?? "");

  // Active deadline toggle
  let deadlineEnabled = $state<boolean>(value.activeDeadlineSeconds != null);
  let deadlineSeconds = $state<number>(value.activeDeadlineSeconds ?? 3600);

  // Mount path validation
  let mountPathError = $derived(
    value.mountPath && !value.mountPath.startsWith("/") ? "Mount path must start with /" : ""
  );

  function commit() {
    onchange?.(value);
  }

  function setImage(v: string) {
    value = {...value, image: v};
    commit();
  }

  function setMountPath(v: string) {
    value = {...value, mountPath: v};
    commit();
  }

  function setReadOnly(checked: boolean) {
    value = {...value, readOnly: checked ? true : false};
    commit();
  }

  function clearReadOnly() {
    const {readOnly: _ro, ...rest} = value;
    value = rest;
    commit();
  }

  function setPromptBeforeSpawn(checked: boolean) {
    value = {...value, promptBeforeSpawn: checked ? true : false};
    commit();
  }

  function clearPromptBeforeSpawn() {
    const {promptBeforeSpawn: _p, ...rest} = value;
    value = rest;
    commit();
  }

  function toggleDeadline(enabled: boolean) {
    deadlineEnabled = enabled;
    if (enabled) {
      value = {...value, activeDeadlineSeconds: deadlineSeconds};
    } else {
      const {activeDeadlineSeconds: _d, ...rest} = value;
      value = rest;
    }
    commit();
  }

  function setDeadlineSeconds(n: number) {
    deadlineSeconds = n;
    if (deadlineEnabled) {
      value = {...value, activeDeadlineSeconds: n};
      commit();
    }
  }

  function toggleResources(enabled: boolean) {
    resourcesEnabled = enabled;
    if (enabled) {
      value = {...value, resources: buildResources()};
    } else {
      const {resources: _r, ...rest} = value;
      value = rest;
    }
    commit();
  }

  function buildResources() {
    const requests: Record<string, string> = {};
    const limits: Record<string, string> = {};
    if (cpuRequest) requests.cpu = cpuRequest;
    if (memoryRequest) requests.memory = memoryRequest;
    if (cpuLimit) limits.cpu = cpuLimit;
    if (memoryLimit) limits.memory = memoryLimit;
    return {
      requests: Object.keys(requests).length ? requests : undefined,
      limits: Object.keys(limits).length ? limits : undefined,
    };
  }

  function updateResources() {
    if (!resourcesEnabled) return;
    value = {...value, resources: buildResources()};
    commit();
  }

  function commitNodeSelector() {
    const ns: Record<string, string> = {};
    for (const [k, v] of nodeSelectorPairs) {
      if (k) ns[k] = v;
    }
    const existing = value.nodeSelector ?? {};
    const existingKeys = Object.keys(existing);
    const nextKeys = Object.keys(ns);
    if (nextKeys.length === 0 && existingKeys.length === 0) return;
    if (
      nextKeys.length === existingKeys.length &&
      nextKeys.every((k) => existing[k] === ns[k])
    ) {
      return;
    }
    if (nextKeys.length === 0) {
      const {nodeSelector: _n, ...rest} = value;
      value = rest;
    } else {
      value = {...value, nodeSelector: ns};
    }
    commit();
  }

  $effect(() => {
    // Touch deeply so the effect re-runs on any pair mutation
    void nodeSelectorPairs.length;
    for (const p of nodeSelectorPairs) {
      void p[0];
      void p[1];
    }
    commitNodeSelector();
  });

  function parseTolerations() {
    const txt = tolerationsText.trim();
    if (!txt) {
      tolerationsError = "";
      const {tolerations: _t, ...rest} = value;
      value = rest;
      commit();
      return;
    }
    try {
      const parsed = JSON.parse(txt);
      if (!Array.isArray(parsed)) {
        tolerationsError = "Tolerations must be a JSON array";
        return;
      }
      tolerationsError = "";
      value = {...value, tolerations: parsed as Record<string, unknown>[]};
      commit();
    } catch (e) {
      tolerationsError = `Invalid JSON: ${e instanceof Error ? e.message : String(e)}`;
    }
  }

  function setOrphanCleanup(v: string) {
    value = {...value, orphanCleanupOnStartup: v};
    commit();
  }

  let orphanCleanup = $state<string>(value.orphanCleanupOnStartup ?? "prompt");
  $effect(() => {
    if (orphanCleanup !== (value.orphanCleanupOnStartup ?? "prompt")) {
      setOrphanCleanup(orphanCleanup);
    }
  });
</script>

<div class="space-y-6">
  <div>
    <label class="block text-sm font-medium text-fg mb-1">
      Image
      <input
        type="text"
        value={value.image ?? ""}
        oninput={(e) => setImage((e.target as HTMLInputElement).value)}
        placeholder="alpine:edge"
        class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
      >
    </label>
  </div>

  <div>
    <label class="block text-sm font-medium text-fg mb-1">
      Mount Path
      <input
        type="text"
        value={value.mountPath ?? ""}
        oninput={(e) => setMountPath((e.target as HTMLInputElement).value)}
        placeholder="/mnt/volume"
        class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        aria-invalid={mountPathError ? "true" : "false"}
      >
    </label>
    {#if mountPathError}
      <p class="text-xs text-destructive mt-1">{mountPathError}</p>
    {/if}
  </div>

  <div>
    <span class="block text-sm font-medium text-fg mb-2">Read-only</span>
    <div class="space-y-2">
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={value.readOnly != null}
          onchange={(e) => {
            const c = (e.target as HTMLInputElement).checked;
            if (c) setReadOnly(true); else clearReadOnly();
          }}
          class="accent-accent"
        >
        <span class="text-sm text-fg">Set read-only explicitly</span>
      </label>
      {#if value.readOnly != null}
        <label class="flex items-center gap-2 cursor-pointer ml-6">
          <input
            type="checkbox"
            checked={value.readOnly ?? false}
            onchange={(e) => setReadOnly((e.target as HTMLInputElement).checked)}
            class="accent-accent"
          >
          <span class="text-sm text-fg">Mount read-only</span>
        </label>
      {/if}
    </div>
  </div>

  <div>
    <span class="block text-sm font-medium text-fg mb-2">Active deadline</span>
    <label class="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        checked={deadlineEnabled}
        onchange={(e) => toggleDeadline((e.target as HTMLInputElement).checked)}
        class="accent-accent"
      >
      <span class="text-sm text-fg">Kill after N seconds</span>
    </label>
    {#if deadlineEnabled}
      <input
        type="number"
        min="1"
        value={deadlineSeconds}
        oninput={(e) => setDeadlineSeconds(Number((e.target as HTMLInputElement).value))}
        class="mt-2 ml-6 w-40 px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
      >
    {/if}
  </div>

  <div>
    <span class="block text-sm font-medium text-fg mb-2">Container resources</span>
    <label class="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        checked={resourcesEnabled}
        onchange={(e) => toggleResources((e.target as HTMLInputElement).checked)}
        class="accent-accent"
      >
      <span class="text-sm text-fg">Set container resources</span>
    </label>
    {#if resourcesEnabled}
      <div class="mt-2 ml-6 grid grid-cols-2 gap-3">
        <label class="block text-xs text-muted">
          CPU request
          <input
            type="text"
            value={cpuRequest}
            oninput={(e) => { cpuRequest = (e.target as HTMLInputElement).value; updateResources(); }}
            placeholder="10m"
            class="w-full px-2 py-1 rounded border border-border bg-surface text-fg text-sm"
          >
        </label>
        <label class="block text-xs text-muted">
          CPU limit
          <input
            type="text"
            value={cpuLimit}
            oninput={(e) => { cpuLimit = (e.target as HTMLInputElement).value; updateResources(); }}
            placeholder="100m"
            class="w-full px-2 py-1 rounded border border-border bg-surface text-fg text-sm"
          >
        </label>
        <label class="block text-xs text-muted">
          Memory request
          <input
            type="text"
            value={memoryRequest}
            oninput={(e) => { memoryRequest = (e.target as HTMLInputElement).value; updateResources(); }}
            placeholder="32Mi"
            class="w-full px-2 py-1 rounded border border-border bg-surface text-fg text-sm"
          >
        </label>
        <label class="block text-xs text-muted">
          Memory limit
          <input
            type="text"
            value={memoryLimit}
            oninput={(e) => { memoryLimit = (e.target as HTMLInputElement).value; updateResources(); }}
            placeholder="128Mi"
            class="w-full px-2 py-1 rounded border border-border bg-surface text-fg text-sm"
          >
        </label>
      </div>
    {/if}
  </div>

  <div>
    <span class="block text-sm font-medium text-fg mb-2">Node selector</span>
    <KeyValuePairEditor bind:pairs={nodeSelectorPairs} keyPlaceholder="key" valuePlaceholder="value" />
  </div>

  <div>
    <label class="block text-sm font-medium text-fg mb-1">
      Tolerations (JSON array)
      <textarea
        rows="4"
        bind:value={tolerationsText}
        onblur={parseTolerations}
        placeholder={'[{"key":"dedicated","operator":"Equal","value":"gpu","effect":"NoSchedule"}]'}
        class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-xs font-mono"
      ></textarea>
    </label>
    {#if tolerationsError}
      <p class="text-xs text-destructive mt-1">{tolerationsError}</p>
    {/if}
  </div>

  <div>
    <span class="block text-sm font-medium text-fg mb-2">Prompt before spawn</span>
    <label class="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        checked={value.promptBeforeSpawn != null}
        onchange={(e) => {
          const c = (e.target as HTMLInputElement).checked;
          if (c) setPromptBeforeSpawn(true); else clearPromptBeforeSpawn();
        }}
        class="accent-accent"
      >
      <span class="text-sm text-fg">Set prompt behavior explicitly</span>
    </label>
    {#if value.promptBeforeSpawn != null}
      <label class="flex items-center gap-2 cursor-pointer ml-6 mt-2">
        <input
          type="checkbox"
          checked={value.promptBeforeSpawn ?? false}
          onchange={(e) => setPromptBeforeSpawn((e.target as HTMLInputElement).checked)}
          class="accent-accent"
        >
        <span class="text-sm text-fg">Prompt before spawning volume browser</span>
      </label>
    {/if}
  </div>

  <div>
    <span class="block text-sm font-medium text-fg mb-2">Orphan cleanup on startup</span>
    <div class="w-48">
      <Select
        bind:value={orphanCleanup}
        options={[
          {value: "prompt", label: "Prompt"},
          {value: "auto", label: "Auto-delete"},
          {value: "ignore", label: "Ignore"},
        ]}
      />
    </div>
  </div>
</div>
