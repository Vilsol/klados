<script lang="ts">
  import { dump as yamlDump } from "js-yaml";
  import { getLastAppliedConfig, stripServerFields } from "../../kubernetes/metadata";
  import { DiffView } from "@klados/ui";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let lastApplied = $derived(getLastAppliedConfig(obj));

  let currentYaml = $derived(yamlDump(stripServerFields(obj as Record<string, any>)));
  let lastAppliedYaml = $derived(
    lastApplied ? yamlDump(stripServerFields(lastApplied)) : ""
  );
</script>

{#if !lastApplied}
  <div class="p-4 text-muted text-sm">
    No <code>last-applied-configuration</code> annotation on this resource.
    Drift detection is only available for resources managed with <code>kubectl apply</code>.
  </div>
{:else}
  <div class="h-full">
    <DiffView
      original={lastAppliedYaml}
      modified={currentYaml}
      mode="split"
    />
  </div>
{/if}
