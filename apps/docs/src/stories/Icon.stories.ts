import type { Meta, StoryObj } from '@storybook/svelte'
import IconStory from './IconStory.svelte'

const meta = {
  title: 'Icon',
  component: IconStory,
  argTypes: {
    iconName: {
      control: 'select',
      options: ['Server', 'AlertCircle', 'CheckCircle', 'Settings', 'Trash2', 'Plus', 'RefreshCw'],
    },
  },
} satisfies Meta<typeof IconStory>

export default meta
type Story = StoryObj<typeof meta>

export const Small: Story = {
  args: { iconName: 'Server', size: 16 },
}

export const Large: Story = {
  args: { iconName: 'Server', size: 32 },
}

export const Alert: Story = {
  args: { iconName: 'AlertCircle', size: 20 },
}
