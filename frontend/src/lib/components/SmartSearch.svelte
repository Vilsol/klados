<script lang="ts">
  import { parseSearch, type SearchTerm } from '$lib/search/parser'
  import { getSuggestions, type Suggestion } from '$lib/search/autocomplete'
  import SmartSearchAutocomplete from './SmartSearchAutocomplete.svelte'

  let {
    items = [],
    value = $bindable(''),
    ontermschange,
  }: {
    items: Record<string, any>[]
    value?: string
    ontermschange?: (terms: SearchTerm[]) => void
  } = $props()

  let inputEl: HTMLInputElement | undefined = $state()
  let suggestions = $state<Suggestion[]>([])
  let selectedIndex = $state(0)
  let showAutocomplete = $state(false)

  let terms = $derived.by(() => {
    return parseSearch(value)
  })

  // Chips are completed terms (all except the trailing incomplete token)
  let chips = $derived.by(() => {
    const raw = value
    if (!raw.trim()) return []
    // If input ends with a space, all tokens are complete
    if (raw.endsWith(' ')) return terms
    // Otherwise, last token is still being typed — exclude it from chips
    return terms.slice(0, -1)
  })

  $effect(() => {
    ontermschange?.(terms)
  })

  function updateAutocomplete() {
    if (!inputEl) return
    const cursor = inputEl.selectionStart ?? value.length
    suggestions = getSuggestions(value, cursor, items)
    selectedIndex = 0
    showAutocomplete = suggestions.length > 0
  }

  function handleInput() {
    updateAutocomplete()
  }

  function handleFocus() {
    updateAutocomplete()
  }

  function handleBlur() {
    showAutocomplete = false
  }

  function applySuggestion(suggestion: Suggestion) {
    const cursor = inputEl?.selectionStart ?? value.length
    const before = value.substring(0, cursor)
    const after = value.substring(cursor)
    const lastSpace = before.lastIndexOf(' ')
    const tokenStart = before.substring(lastSpace + 1)

    // Strip negation prefix to find the qualifier part
    const stripped = tokenStart.startsWith('-') ? tokenStart.substring(1) : tokenStart
    const negPrefix = tokenStart.startsWith('-') ? '-' : ''
    const colonIdx = stripped.indexOf(':')

    let replacement: string

    if (colonIdx === -1) {
      // Suggesting a qualifier — replace the partial text with the qualifier
      replacement = negPrefix + suggestion.value
    } else {
      // Suggesting a key or value after qualifier
      const qualifier = stripped.substring(0, colonIdx + 1)
      const afterColon = stripped.substring(colonIdx + 1)
      const eqIdx = afterColon.indexOf('=')

      if (eqIdx === -1) {
        // Suggesting a key — append = to invite value completion
        replacement = negPrefix + qualifier + suggestion.value + '='
      } else {
        // Suggesting a value — complete the value and add space
        const key = afterColon.substring(0, eqIdx)
        replacement = negPrefix + qualifier + key + '=' + suggestion.value + ' '
      }
    }

    value = before.substring(0, lastSpace + 1) + replacement + after
    showAutocomplete = false

    // Re-focus and move cursor to end of replacement
    requestAnimationFrame(() => {
      inputEl?.focus()
      updateAutocomplete()
    })
  }

  function handleKeydown(e: KeyboardEvent) {
    if (showAutocomplete && suggestions.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        selectedIndex = (selectedIndex + 1) % suggestions.length
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        selectedIndex = (selectedIndex - 1 + suggestions.length) % suggestions.length
        return
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        applySuggestion(suggestions[selectedIndex])
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        showAutocomplete = false
        return
      }
    }
  }

  function removeChip(index: number) {
    const allTerms = [...terms]
    allTerms.splice(index, 1)
    // Rebuild input from remaining terms + trailing text
    const parts = allTerms.map((t) => {
      const neg = t.negated ? '-' : ''
      if (t.type === 'text' || t.type === 'phrase') {
        return t.type === 'phrase' ? `${neg}"${t.value}"` : `${neg}${t.value}`
      }
      return `${neg}${t.type}:${t.value}`
    })
    value = parts.join(' ') + (parts.length > 0 ? ' ' : '')
    requestAnimationFrame(() => inputEl?.focus())
  }

  function chipColor(type: string): string {
    switch (type) {
      case 'label': return 'bg-blue-500/15 text-blue-400 border-blue-500/30'
      case 'annotation': return 'bg-purple-500/15 text-purple-400 border-purple-500/30'
      case 'namespace': return 'bg-green-500/15 text-green-400 border-green-500/30'
      case 'name': return 'bg-orange-500/15 text-orange-400 border-orange-500/30'
      default: return 'bg-muted/15 text-fg border-border'
    }
  }

  function chipLabel(term: SearchTerm): string {
    const neg = term.negated ? '-' : ''
    if (term.type === 'text' || term.type === 'phrase') {
      return `${neg}${term.value}`
    }
    const short: Record<string, string> = { label: 'l', annotation: 'ann', namespace: 'ns', name: 'n' }
    return `${neg}${short[term.type] ?? term.type}:${term.value}`
  }
</script>

<div class="relative flex items-center gap-1 flex-1 min-w-0">
  <div class="flex flex-wrap items-center gap-1 flex-1 min-w-0 px-2 py-1 bg-surface border border-border rounded text-sm focus-within:ring-1 focus-within:ring-accent">
    {#each chips as chip, i}
      <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border text-xs font-mono {chipColor(chip.type)} {chip.negated ? 'line-through opacity-75' : ''}">
        {chipLabel(chip)}
        <button
          class="ml-0.5 hover:text-fg"
          onclick={() => removeChip(i)}
          tabindex={-1}
        >&times;</button>
      </span>
    {/each}
    <input
      bind:this={inputEl}
      bind:value={value}
      oninput={handleInput}
      onfocus={handleFocus}
      onblur={handleBlur}
      onkeydown={handleKeydown}
      class="flex-1 min-w-24 bg-transparent outline-none text-fg placeholder:text-muted"
      placeholder={chips.length === 0 ? 'Filter resources... (label:key=value, name:..., ns:...)' : ''}
    />
  </div>

  <SmartSearchAutocomplete
    {suggestions}
    visible={showAutocomplete}
    {selectedIndex}
    onselect={applySuggestion}
  />
</div>
