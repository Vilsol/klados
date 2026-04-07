import type { Meta, StoryObj } from '@storybook/svelte'
import { Select } from '@klados/ui'

const meta = {
  title: 'Select',
  component: Select,
} satisfies Meta<typeof Select>

export default meta
type Story = StoryObj<typeof meta>

const namespaceOptions = [
  { value: 'default', label: 'default' },
  { value: 'kube-system', label: 'kube-system' },
  { value: 'monitoring', label: 'monitoring' },
  { value: 'production', label: 'production' },
]

export const Default: Story = {
  args: { options: namespaceOptions, value: 'default', size: 'sm' },
}

export const ExtraSmall: Story = {
  args: { options: namespaceOptions, value: 'kube-system', size: 'xs' },
}

export const SingleOption: Story = {
  args: {
    options: [{ value: 'only', label: 'Only option' }],
    value: 'only',
    size: 'sm',
  },
}
