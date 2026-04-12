<script lang="ts">
  import {Combobox} from "@klados/ui";
  import type {KubernetesResource} from "$lib/types";

  interface Props {
    schema: KubernetesResource;
    values: Record<string, unknown>;
    onchange: (key: string, value: unknown) => void;
  }

  let {schema, values, onchange}: Props = $props();

  let properties = $derived(schema?.properties ?? {});
  let propertyKeys = $derived(Object.keys(properties));

  function defaultForType(type: string): unknown {
    if (type === "boolean") {
      return false;
    }
    if (type === "number" || type === "integer") {
      return 0;
    }
    return "";
  }
</script>

<div class="space-y-4">
  {#each propertyKeys as key}
    {@const prop = properties[key]}
    {@const label = prop.title ?? key}
    {@const desc = prop.description}
    {@const value = values[key] ?? prop.default ?? defaultForType(prop.type)}

    <div>
      {#if prop.type === 'boolean'}
        <label class="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={value}
            onchange={(e) => onchange(key, (e.target as HTMLInputElement).checked)}
            class="accent-accent"
          >
          <span class="text-sm text-fg">{label}</span>
        </label>
      {:else if prop.type === 'string' && prop.enum}
        <!-- svelte-ignore a11y_label_has_associated_control -->
        <label class="block text-sm font-medium text-fg mb-1"
          >{label}
          <div class="w-full max-w-xs">
            <Combobox
              options={prop.enum.map((opt: string) => ({ value: opt, label: opt }))}
              {value}
              placeholder="Select…"
              searchPlaceholder="Search…"
              size="sm"
              onValueChange={(v: string) => onchange(key, v)}
            />
          </div>
        </label>
      {:else if prop.type === 'string' && prop.format === 'color'}
        <label class="block text-sm font-medium text-fg mb-1"
          >{label}
          <input
            type="color"
            value={value || '#000000'}
            oninput={(e) => onchange(key, (e.target as HTMLInputElement).value)}
            class="w-8 h-8 rounded cursor-pointer border border-border"
          >
        </label>
      {:else if prop.type === 'number' || prop.type === 'integer'}
        <label class="block text-sm font-medium text-fg mb-1"
          >{label}
          <input
            type="number"
            {value}
            min={prop.minimum}
            max={prop.maximum}
            step={prop.type === 'integer' ? 1 : undefined}
            oninput={(e) => onchange(key, Number((e.target as HTMLInputElement).value))}
            class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          >
        </label>
      {:else}
        <label class="block text-sm font-medium text-fg mb-1"
          >{label}
          <input
            type="text"
            {value}
            oninput={(e) => onchange(key, (e.target as HTMLInputElement).value)}
            class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          >
        </label>
      {/if}
      {#if desc}
        <p class="text-xs text-muted-foreground mt-1">{desc}</p>
      {/if}
    </div>
  {/each}
</div>
