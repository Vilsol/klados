<script lang="ts">
  import {onMount} from "svelte";
  import * as PluginService from "../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js";
  import SchemaForm from "$lib/components/SchemaForm.svelte";
  import {getLogger} from "$lib/logger";

  const log = getLogger("settings");

  interface Props {
    pluginName: string;
  }

  let {pluginName}: Props = $props();

  let schema = $state<any>(null);
  let values = $state<Record<string, any>>({});
  let loading = $state<boolean>(true);

  onMount(() => {
    (async () => {
      try {
        const schemaStr = await PluginService.GetPluginSettingsSchema(pluginName);
        if (schemaStr) {
          schema = JSON.parse(schemaStr);
        }
        const settingsStr = await PluginService.GetPluginSettings(pluginName);
        if (settingsStr) {
          values = JSON.parse(settingsStr);
        }
      } catch (e) {
        log.error("Failed to load plugin settings", {error: String(e)});
      } finally {
        loading = false;
      }
    })();
  });

  function handleChange(key: string, value: any) {
    values = {...values, [key]: value};
    PluginService.SetPluginSettings(pluginName, JSON.stringify(values));
  }
</script>

<div class="max-w-2xl space-y-6">
  <h2 class="text-base font-medium text-fg mb-4">{pluginName}</h2>

  {#if loading}
    <p class="text-sm text-muted-foreground">Loading settings...</p>
  {:else if !schema}
    <p class="text-sm text-muted-foreground">This plugin has no configurable settings.</p>
  {:else}
    <SchemaForm {schema} {values} onchange={handleChange} />
  {/if}
</div>
