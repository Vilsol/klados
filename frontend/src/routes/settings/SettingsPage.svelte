<script lang="ts">
  import SettingsLayout from "./SettingsLayout.svelte";
  import GeneralSettings from "./GeneralSettings.svelte";
  import AppearanceSettings from "./AppearanceSettings.svelte";
  import ClusterListSettings from "./ClusterListSettings.svelte";
  import ClusterSettings from "./ClusterSettings.svelte";
  import KeybindingSettings from "./KeybindingSettings.svelte";
  import FilterSettings from "./FilterSettings.svelte";
  import ColumnSettings from "./ColumnSettings.svelte";
  import PluginListSettings from "./PluginListSettings.svelte";
  import PluginSettings from "./PluginSettings.svelte";

  interface Props {
    params?: {wild?: string};
  }

  let {params}: Props = $props();

  const wild = $derived(params?.wild ?? "");

  const activeSection = $derived((): string => {
    if (!wild) {
      return "general";
    }
    const parts = wild.split("/");
    if (parts[0] === "plugins" && parts[1]) {
      return `plugins/${parts[1]}`;
    }
    return parts[0];
  });
</script>

<SettingsLayout active={activeSection()}>
  {#if !wild}
    <GeneralSettings />
  {:else if wild === 'appearance'}
    <AppearanceSettings />
  {:else if wild === 'clusters'}
    <ClusterListSettings />
  {:else if wild.startsWith('clusters/')}
    <ClusterSettings ctxName={wild.slice('clusters/'.length)} />
  {:else if wild === 'keybindings'}
    <KeybindingSettings />
  {:else if wild === 'filters'}
    <FilterSettings />
  {:else if wild === 'columns'}
    <ColumnSettings />
  {:else if wild === 'plugins'}
    <PluginListSettings />
  {:else if wild.startsWith('plugins/')}
    <PluginSettings pluginName={wild.slice('plugins/'.length)} />
  {:else}
    <GeneralSettings />
  {/if}
</SettingsLayout>
