<script lang="ts">
  import { YAMLEditor } from '@klados/ui'

  let {
    kind = 'Deployment',
    readOnly = false,
  }: {
    kind?: string
    readOnly?: boolean
  } = $props()

  let obj = $state({
    apiVersion: 'apps/v1',
    kind,
    metadata: {
      name: 'my-app',
      namespace: 'default',
      labels: { app: 'my-app' },
    },
    spec: {
      replicas: 3,
      selector: { matchLabels: { app: 'my-app' } },
      template: {
        metadata: { labels: { app: 'my-app' } },
        spec: {
          containers: [{ name: 'app', image: 'nginx:latest', ports: [{ containerPort: 80 }] }],
        },
      },
    },
  })
</script>

<div class="h-96 border border-border rounded overflow-hidden">
  <YAMLEditor
    bind:obj
    ctxName="prod"
    gvr="apps.v1.deployments"
    namespace="default"
    name="my-app"
    {kind}
  />
</div>
