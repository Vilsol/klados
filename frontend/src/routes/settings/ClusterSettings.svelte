<script lang="ts">
  interface Props {
    ctxName: string
  }

  let { ctxName }: Props = $props()
</script>

<div class="text-fg">Cluster: {ctxName}</div>
