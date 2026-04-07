import type { Meta, StoryObj } from '@storybook/svelte'
import { CodeBlock } from '@klados/ui'

const meta = {
  title: 'CodeBlock',
  component: CodeBlock,
} satisfies Meta<typeof CodeBlock>

export default meta
type Story = StoryObj<typeof meta>

export const JSON: Story = {
  args: {
    lang: 'json',
    value: `{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "my-app",
    "namespace": "default"
  }
}`,
  },
}

export const YAML: Story = {
  args: {
    lang: 'yaml',
    value: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app`,
  },
}

export const Shell: Story = {
  args: {
    lang: 'shell',
    value: `kubectl get pods -n default\nkubectl describe pod my-pod\nkubectl logs my-pod --follow`,
  },
}
