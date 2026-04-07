import type { Meta, StoryObj } from '@storybook/svelte'
import DialogStory from './DialogStory.svelte'

const meta = {
  title: 'Dialog',
  component: DialogStory,
} satisfies Meta<typeof DialogStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    title: 'Confirm action',
    description: 'This will apply changes to your cluster.',
    showFooter: false,
  },
}

export const WithFooter: Story = {
  args: {
    title: 'Apply changes',
    description: 'Review the changes before applying.',
    showFooter: true,
  },
}
