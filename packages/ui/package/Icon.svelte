<script lang="ts">
  import type { Component } from 'svelte'

  let {
    icon: IconComponent,
    size = 16,
    class: className = '',
  }: {
    icon: Component<{ size?: number; class?: string }>
    size?: number
    class?: string
  } = $props()
</script>

<IconComponent {size} class={className} />
