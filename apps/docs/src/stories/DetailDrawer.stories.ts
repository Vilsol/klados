import type { Meta, StoryObj } from '@storybook/svelte'
import DetailDrawerStory from './DetailDrawerStory.svelte'

const meta = {
  title: 'DetailDrawer',
  component: DetailDrawerStory,
} satisfies Meta<typeof DetailDrawerStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    resourceName: 'nginx-abc123',
    resourceNamespace: 'default',
    gvr: 'core.v1.pods',
  },
}

export const WithSections: Story = {
  args: {
    resourceName: 'my-deployment',
    resourceNamespace: 'production',
    gvr: 'apps.v1.deployments',
  },
}
