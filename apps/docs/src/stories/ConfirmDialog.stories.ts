import type { Meta, StoryObj } from '@storybook/svelte'
import ConfirmDialogStory from './ConfirmDialogStory.svelte'

const meta = {
  title: 'ConfirmDialog',
  component: ConfirmDialogStory,
} satisfies Meta<typeof ConfirmDialogStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    title: 'Confirm action',
    message: 'This will apply changes to the cluster. Proceed?',
    confirmLabel: 'Apply',
    isDestructive: false,
  },
}

export const Destructive: Story = {
  args: {
    title: 'Delete resource',
    message: 'This will permanently delete the pod. This action cannot be undone.',
    confirmLabel: 'Delete',
    isDestructive: true,
  },
}
