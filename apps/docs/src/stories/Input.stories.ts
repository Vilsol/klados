import type { Meta, StoryObj } from '@storybook/svelte'
import { Input } from '@klados/ui'

const meta = {
  title: 'Input',
  component: Input,
} satisfies Meta<typeof Input>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { label: 'Namespace', placeholder: 'default' },
}

export const WithError: Story = {
  args: { label: 'Cluster Name', placeholder: 'my-cluster', error: 'Name is required' },
}

export const Disabled: Story = {
  args: { label: 'Read-only field', value: 'fixed-value', disabled: true },
}
