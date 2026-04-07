import type { Meta, StoryObj } from '@storybook/svelte'
import ButtonStory from './ButtonStory.svelte'

const meta = {
  title: 'Button',
  component: ButtonStory,
  argTypes: {
    variant: { control: 'select', options: ['primary', 'destructive', 'ghost', 'outline'] },
  },
} satisfies Meta<typeof ButtonStory>

export default meta
type Story = StoryObj<typeof meta>

export const Primary: Story = {
  args: { variant: 'primary', label: 'Click me' },
}

export const Destructive: Story = {
  args: { variant: 'destructive', label: 'Delete' },
}

export const Ghost: Story = {
  args: { variant: 'ghost', label: 'Cancel' },
}

export const Outline: Story = {
  args: { variant: 'outline', label: 'View details' },
}

export const Disabled: Story = {
  args: { variant: 'primary', label: 'Disabled', disabled: true },
}
