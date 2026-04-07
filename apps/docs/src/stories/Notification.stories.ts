import type { Meta, StoryObj } from '@storybook/svelte'
import NotificationStory from './NotificationStory.svelte'

const meta = {
  title: 'Notification',
  component: NotificationStory,
  argTypes: {
    type: { control: 'select', options: ['info', 'success', 'error'] },
  },
} satisfies Meta<typeof NotificationStory>

export default meta
type Story = StoryObj<typeof meta>

export const Info: Story = {
  args: { type: 'info', message: 'Deployment scaled to 3 replicas', withDetails: false },
}

export const Error: Story = {
  args: { type: 'error', message: 'Failed to connect to cluster', withDetails: true },
}

export const SuccessToast: Story = {
  args: { type: 'success', message: 'Changes applied successfully', withDetails: false },
}
