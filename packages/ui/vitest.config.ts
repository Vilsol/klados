import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'

export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  resolve: {
    dedupe: ['svelte'],
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    setupFiles: ['src/lib/__tests__/setup.ts'],
    server: {
      deps: {
        inline: ['bits-ui', 'lucide-svelte', 'runed'],
      },
    },
  },
})
