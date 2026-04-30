import type { Meta, StoryObj } from '@storybook/svelte'
import VirtualLogViewerStory from './VirtualLogViewerStory.svelte'

const meta = {
  title: 'VirtualLogViewer',
  component: VirtualLogViewerStory,
} satisfies Meta<typeof VirtualLogViewerStory>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {
  args: { lineCount: 0, includeErrors: false, showTimestamps: false },
}

export const WithLines: Story = {
  args: { lineCount: 50, includeErrors: true, showTimestamps: false },
}

export const WithTimestamps: Story = {
  args: { lineCount: 30, includeErrors: false, showTimestamps: true },
}

export const LongLines: Story = {
  args: { lineCount: 30, longLines: true, includeErrors: true },
}

export const ManyMatches: Story = {
  args: { lineCount: 80, manyMatches: true, includeErrors: true },
}

export const Tall: Story = {
  args: { lineCount: 1000, includeErrors: true },
}
