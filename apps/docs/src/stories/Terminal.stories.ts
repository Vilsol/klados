import type { Meta, StoryObj } from '@storybook/svelte'
import TerminalStory from './TerminalStory.svelte'

const meta = {
  title: 'Terminal',
  component: TerminalStory,
} satisfies Meta<typeof TerminalStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { sessionID: 'story-session-1' },
}

export const AltSession: Story = {
  args: { sessionID: 'story-session-2' },
}
