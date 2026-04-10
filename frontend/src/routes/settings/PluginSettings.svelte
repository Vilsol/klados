<script lang="ts">
  interface Props {
    pluginName: string
  }

  let { pluginName }: Props = $props()
</script>

<div class="text-fg">Plugin: {pluginName}</div>
