import type { Meta, StoryObj } from '@storybook/svelte'
import TooltipStory from './TooltipStory.svelte'

const meta = {
  title: 'Tooltip',
  component: TooltipStory,
} satisfies Meta<typeof TooltipStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { content: 'This is a tooltip', triggerLabel: 'Hover me' },
}

export const LongContent: Story = {
  args: {
    content: 'This tooltip explains a complex feature in detail',
    triggerLabel: 'More info',
  },
}
