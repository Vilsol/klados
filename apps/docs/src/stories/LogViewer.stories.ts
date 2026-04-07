import type { Meta, StoryObj } from '@storybook/svelte'
import LogViewerStory from './LogViewerStory.svelte'

const meta = {
  title: 'LogViewer',
  component: LogViewerStory,
} satisfies Meta<typeof LogViewerStory>

export default meta
type Story = StoryObj<typeof meta>

export const Loading: Story = {
  args: { streamID: 'story-stream-1' },
}

export const DifferentStream: Story = {
  args: { streamID: 'story-stream-2' },
}
