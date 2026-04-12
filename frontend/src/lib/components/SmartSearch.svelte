<script lang="ts">
  import {X} from "lucide-svelte";
  import {parseSearch, type SearchTerm} from "$lib/search/parser";
  import {getSuggestions, type Suggestion} from "$lib/search/autocomplete";
  import {filterItems} from "$lib/search/filter";
  import {preferencesStore} from "$lib/stores/preferences.svelte";
  import SmartSearchAutocomplete from "./SmartSearchAutocomplete.svelte";
  import type {KubernetesResource} from "$lib/types";

  let {
    items = [],
    value = $bindable(""),
    ontermschange,
  }: {
    items: Record<string, KubernetesResource>[];
    value?: string;
    ontermschange?: (terms: SearchTerm[]) => void;
  } = $props();

  let inputEl: HTMLInputElement | undefined = $state();
  let suggestions = $state<Suggestion[]>([]);
  let selectedIndex = $state(0);
  let showAutocomplete = $state(false);

  // inputText is the visible text in the <input> — only the trailing (incomplete) token
  let inputText = $state("");

  let terms = $derived.by(() => parseSearch(value));

  // Chips are completed terms (all except the trailing incomplete token)
  let chips = $derived.by(() => {
    if (!value.trim()) {
      return [];
    }
    if (value.endsWith(" ")) {
      return terms;
    }
    return terms.slice(0, -1);
  });

  // Rebuild value from chips + trailing input text
  function rebuildValue() {
    const chipParts = chips.map((t) => serializeTerm(t));
    const parts = chipParts.length > 0 ? [...chipParts, inputText] : [inputText];
    value = parts.join(" ");
    // Ensure trailing space after chips if there are chips and no trailing text
    if (chips.length > 0 && !inputText) {
      value = `${chipParts.join(" ")} `;
    }
  }

  // When value is set externally (e.g., saved filter applied), extract trailing text
  let lastExternalValue = "";
  $effect(() => {
    if (value !== lastExternalValue) {
      lastExternalValue = value;
      syncInputText();
    }
  });

  $effect(() => {
    ontermschange?.(terms);
  });

  function updateAutocomplete() {
    if (!inputEl) {
      return;
    }
    const pool = preferencesStore.prefs.contextualAutocomplete ? filterItems(items, chips) : items;
    suggestions = getSuggestions(value, value.length, pool);
    selectedIndex = 0;
    showAutocomplete = suggestions.length > 0;
  }

  function handleInput() {
    rebuildValue();
    syncInputText();
    lastExternalValue = value;
    updateAutocomplete();
  }

  function syncInputText() {
    const raw = value;
    if (!raw.trim()) {
      inputText = "";
    } else if (raw.endsWith(" ")) {
      inputText = "";
    } else {
      const lastSpace = raw.lastIndexOf(" ");
      inputText = raw.substring(lastSpace + 1);
    }
  }

  function handleFocus() {
    updateAutocomplete();
  }

  function handleBlur() {
    showAutocomplete = false;
  }

  function applySuggestion(suggestion: Suggestion) {
    const token = inputText;
    const stripped = token.startsWith("-") ? token.substring(1) : token;
    const negPrefix = token.startsWith("-") ? "-" : "";
    const colonIdx = stripped.indexOf(":");

    let replacement: string;

    if (colonIdx === -1) {
      replacement = negPrefix + suggestion.value;
    } else {
      const qualifier = stripped.substring(0, colonIdx + 1);
      const afterColon = stripped.substring(colonIdx + 1);
      const eqIdx = afterColon.indexOf("=");
      // namespace: and name: take plain values, not key=value
      const isPlainValue = qualifier === "namespace:" || qualifier === "ns:" || qualifier === "name:" || qualifier === "n:";

      if (eqIdx === -1 && isPlainValue) {
        replacement = `${negPrefix + qualifier + suggestion.value} `;
      } else if (eqIdx === -1) {
        replacement = `${negPrefix + qualifier + suggestion.value}=`;
      } else {
        const key = afterColon.substring(0, eqIdx);
        replacement = `${negPrefix + qualifier + key}=${suggestion.value} `;
      }
    }

    inputText = replacement.endsWith(" ") ? "" : replacement;
    // Rebuild with the completed token
    const chipParts = chips.map((t) => serializeTerm(t));
    if (replacement.endsWith(" ")) {
      value = `${[...chipParts, replacement.trimEnd()].join(" ")} `;
    } else {
      value = [...chipParts, replacement].join(" ");
    }
    lastExternalValue = value;
    showAutocomplete = false;

    requestAnimationFrame(() => {
      inputEl?.focus();
      updateAutocomplete();
    });
  }

  function handleKeydown(e: KeyboardEvent) {
    if (showAutocomplete && suggestions.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        selectedIndex = (selectedIndex + 1) % suggestions.length;
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        selectedIndex = (selectedIndex - 1 + suggestions.length) % suggestions.length;
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        applySuggestion(suggestions[selectedIndex]);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        showAutocomplete = false;
        return;
      }
    }
    // Backspace into chips when input is empty
    if (e.key === "Backspace" && inputText === "" && chips.length > 0) {
      e.preventDefault();
      removeChip(chips.length - 1);
    }
  }

  function removeChip(index: number) {
    const remaining = chips.filter((_, i) => i !== index);
    const parts = remaining.map((t) => serializeTerm(t));
    value = parts.length > 0 ? `${parts.join(" ")} ${inputText}` : inputText;
    lastExternalValue = value;
    requestAnimationFrame(() => inputEl?.focus());
  }

  function serializeTerm(t: SearchTerm): string {
    const neg = t.negated ? "-" : "";
    if (t.type === "text") {
      return `${neg}${t.value}`;
    }
    if (t.type === "phrase") {
      return `${neg}"${t.value}"`;
    }
    return `${neg}${t.type}:${t.value}`;
  }

  function clearAll() {
    value = "";
    inputText = "";
    lastExternalValue = "";
    showAutocomplete = false;
    requestAnimationFrame(() => inputEl?.focus());
  }

  function chipColor(type: string): string {
    switch (type) {
      case "label":
        return "bg-blue-500/15 text-blue-400 border-blue-500/30";
      case "annotation":
        return "bg-purple-500/15 text-purple-400 border-purple-500/30";
      case "namespace":
        return "bg-green-500/15 text-green-400 border-green-500/30";
      case "name":
        return "bg-orange-500/15 text-orange-400 border-orange-500/30";
      default:
        return "bg-muted/15 text-fg border-border";
    }
  }

  function chipLabel(term: SearchTerm): string {
    const neg = term.negated ? "-" : "";
    if (term.type === "text" || term.type === "phrase") {
      return `${neg}${term.value}`;
    }
    const short: Record<string, string> = {label: "l", annotation: "ann", namespace: "ns", name: "n"};
    return `${neg}${short[term.type] ?? term.type}:${term.value}`;
  }
</script>

<div class="relative flex items-center gap-1 flex-1 min-w-0">
  <div
    class="flex flex-wrap items-center gap-1 flex-1 min-w-0 px-2 py-1 bg-surface border border-border rounded text-sm focus-within:ring-1 focus-within:ring-accent"
  >
    {#each chips as chip, i}
      <span
        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border text-xs font-mono {chipColor(chip.type)} {chip.negated ? 'line-through opacity-75' : ''}"
      >
        {chipLabel(chip)}
        <button type="button" class="ml-0.5 hover:text-fg" onclick={() => removeChip(i)} tabindex={-1}>&times;</button>
      </span>
    {/each}
    <input
      bind:this={inputEl}
      bind:value={inputText}
      oninput={handleInput}
      onfocus={handleFocus}
      onblur={handleBlur}
      onkeydown={handleKeydown}
      class="flex-1 min-w-24 bg-transparent outline-none text-fg placeholder:text-muted"
      placeholder={chips.length === 0 ? 'Filter resources... (label:key=value, name:..., ns:...)' : ''}
    >
    {#if value.trim()}
      <button
        type="button"
        class="shrink-0 p-0.5 rounded text-muted hover:text-fg hover:bg-surface-hover transition-colors"
        onclick={clearAll}
        tabindex={-1}
        title="Clear filter"
      >
        <X class="w-3.5 h-3.5" />
      </button>
    {/if}
  </div>

  <SmartSearchAutocomplete {suggestions} visible={showAutocomplete} {selectedIndex} onselect={applySuggestion} />
</div>
