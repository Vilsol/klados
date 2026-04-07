import type { Meta, StoryObj } from '@storybook/svelte'
import TabBarStory from './TabBarStory.svelte'

const meta = {
  title: 'TabBar',
  component: TabBarStory,
} satisfies Meta<typeof TabBarStory>

export default meta
type Story = StoryObj<typeof meta>

export const TwoTabs: Story = {
  args: { tabCount: 2 },
}

export const ManyTabs: Story = {
  args: { tabCount: 5 },
}
