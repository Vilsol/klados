import type { Meta, StoryObj } from '@storybook/svelte'
import YAMLEditorStory from './YAMLEditorStory.svelte'

const meta = {
  title: 'YAMLEditor',
  component: YAMLEditorStory,
} satisfies Meta<typeof YAMLEditorStory>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { kind: 'Deployment', readOnly: false },
}

export const Service: Story = {
  args: { kind: 'Service', readOnly: false },
}
