import type { Meta, StoryObj } from '@storybook/svelte'
import DiffViewStory from './DiffViewStory.svelte'

const original = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  labels:
    app: my-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: app
          image: nginx:1.24
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
`

const modified = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  labels:
    app: my-app
    version: v2
spec:
  replicas: 5
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
        version: v2
    spec:
      containers:
        - name: app
          image: nginx:1.25
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: 200m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
`

const meta = {
  title: 'DiffView',
  component: DiffViewStory,
} satisfies Meta<typeof DiffViewStory>

export default meta
type Story = StoryObj<typeof meta>

export const SideBySide: Story = {
  args: { original, modified, mode: 'split' },
}

export const Unified: Story = {
  args: { original, modified, mode: 'unified' },
}
