import type { Meta, StoryObj } from '@storybook/svelte'
import BadgeStory from './BadgeStory.svelte'

const meta = {
  title: 'Badge',
  component: BadgeStory,
  argTypes: {
    variant: { control: 'select', options: ['default', 'success', 'warning', 'destructive'] },
  },
} satisfies Meta<typeof BadgeStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { variant: 'default', text: 'Pending' },
}

export const Success: Story = {
  args: { variant: 'success', text: 'Running' },
}

export const Warning: Story = {
  args: { variant: 'warning', text: 'Degraded' },
}

export const Destructive: Story = {
  args: { variant: 'destructive', text: 'Failed' },
}
