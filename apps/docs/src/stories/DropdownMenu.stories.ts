import type { Meta, StoryObj } from '@storybook/svelte'
import DropdownMenuStory from './DropdownMenuStory.svelte'

const meta = {
  title: 'DropdownMenu',
  component: DropdownMenuStory,
} satisfies Meta<typeof DropdownMenuStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { withIcons: false },
}

export const WithIcons: Story = {
  args: { withIcons: true },
}
