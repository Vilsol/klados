<script lang="ts">
  import { DiffView } from '@klados/ui'

  let {
    original,
    modified,
    mode = 'split',
  }: {
    original: string
    modified: string
    mode?: 'split' | 'unified'
  } = $props()
</script>

<DiffView {original} {modified} {mode} />
