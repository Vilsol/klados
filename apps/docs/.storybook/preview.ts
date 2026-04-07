import '../src/app.css'
import '@klados/ui/theme.css'
import type { Preview } from '@storybook/svelte'

const preview: Preview = {
  parameters: {
    backgrounds: { disable: true },
  },
}

export default preview
