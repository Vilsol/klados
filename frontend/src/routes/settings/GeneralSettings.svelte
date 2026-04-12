<script lang="ts">
  import {onMount} from "svelte";
  import * as ConfigService from "../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js";
  import {preferencesStore} from "$lib/stores/preferences.svelte";

  let theme = $state<string>("system");
  let fontSize = $state<number>(14);
  let startupBehavior = $state<string>("last");
  let startupCluster = $state<string>("");
  let terminalWebGL = $state<boolean>(false);

  onMount(() => {
    (async () => {
      const config = await ConfigService.GetConfig();
      if (config) {
        theme = (config as any).theme || "system";
        fontSize = (config as any).fontSize || 14;
        startupBehavior = (config as any).startupBehavior || "last";
        startupCluster = (config as any).startupCluster || "";
        terminalWebGL = (config as any).terminalWebGL ?? false;
      }
    })();
  });

  function setTheme(value: string) {
    theme = value;
    ConfigService.SetTheme(value);
  }

  function setFontSize(value: number) {
    fontSize = value;
    ConfigService.SetFontSize(value);
  }

  function setStartup(behavior: string) {
    startupBehavior = behavior;
    ConfigService.SetStartupBehavior(behavior, startupCluster);
  }

  function setStartupCluster(cluster: string) {
    startupCluster = cluster;
    ConfigService.SetStartupBehavior(startupBehavior, cluster);
  }

  function setTerminalWebGL(enabled: boolean) {
    terminalWebGL = enabled;
    ConfigService.SetTerminalWebGL(enabled);
  }
</script>

<div class="max-w-2xl space-y-8">
  <div>
    <h2 class="text-base font-medium text-fg mb-4">Theme</h2>
    <div class="flex gap-2">
      {#each [['system', 'System'], ['light', 'Light'], ['dark', 'Dark']] as [ value, label ]}
        <button
          class="px-3 py-1.5 rounded text-sm {theme === value ? 'bg-accent text-accent-foreground' : 'border border-border text-fg hover:bg-surface-hover'}"
          onclick={() => setTheme(value)}
        >
          {label}
        </button>
      {/each}
    </div>
  </div>

  <div>
    <h2 class="text-base font-medium text-fg mb-4">Font Size</h2>
    <div class="flex items-center gap-4">
      <input
        type="range"
        min="10"
        max="24"
        step="1"
        value={fontSize}
        oninput={(e) => setFontSize(Number((e.target as HTMLInputElement).value))}
        class="flex-1 accent-accent"
      >
      <span class="text-sm text-fg w-8 text-right">{fontSize}px</span>
    </div>
  </div>

  <div>
    <h2 class="text-base font-medium text-fg mb-4">Startup Behavior</h2>
    <div class="space-y-2">
      {#each [['last', 'Reconnect to last session'], ['chooser', 'Show cluster chooser'], ['specific', 'Connect to specific cluster']] as [ value, label ]}
        <label class="flex items-center gap-2 cursor-pointer">
          <input type="radio" name="startup" checked={startupBehavior === value} onchange={() => setStartup(value)} class="accent-accent">
          <span class="text-sm text-fg">{label}</span>
        </label>
      {/each}

      {#if startupBehavior === 'specific'}
        <input
          type="text"
          value={startupCluster}
          oninput={(e) => setStartupCluster((e.target as HTMLInputElement).value)}
          placeholder="Cluster context name"
          class="mt-2 w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        >
      {/if}
    </div>
  </div>

  <div>
    <h2 class="text-base font-medium text-fg mb-4">Terminal</h2>
    <label class="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        checked={terminalWebGL}
        onchange={(e) => setTerminalWebGL((e.target as HTMLInputElement).checked)}
        class="accent-accent"
      >
      <span class="text-sm text-fg">Use WebGL renderer for terminal</span>
    </label>
    <p class="text-sm text-muted-foreground mt-1">May improve performance for terminal output. Requires restart.</p>
  </div>
</div>
