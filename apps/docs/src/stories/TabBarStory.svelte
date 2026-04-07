<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { TabBar, sessionStore } from '@klados/ui'

  let {
    tabCount = 2,
  }: {
    tabCount?: number
  } = $props()

  onMount(() => {
    sessionStore.restore([], 0, false)
    sessionStore.openTab({ clusterContext: 'prod', gvr: 'core.v1.pods', namespace: 'default', name: 'nginx-abc123' })
    if (tabCount > 1) {
      sessionStore.openTab({ clusterContext: 'prod', gvr: 'apps.v1.deployments', namespace: 'default', name: 'my-app' })
    }
    if (tabCount > 2) {
      sessionStore.openTab({ clusterContext: 'prod', gvr: 'core.v1.services', namespace: 'default', name: 'my-svc' })
      sessionStore.openTab({ clusterContext: 'prod', gvr: 'networking.k8s.io.v1.ingresses', namespace: 'default', name: 'my-ingress' })
      sessionStore.openTab({ clusterContext: 'prod', gvr: 'batch.v1.jobs', namespace: 'default', name: 'cron-task' })
    }
  })

  onDestroy(() => {
    sessionStore.restore([], 0, false)
  })
</script>

<TabBar />
