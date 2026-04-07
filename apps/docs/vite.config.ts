import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [tailwindcss()],
  resolve: {
    dedupe: ['svelte'],
    conditions: ['svelte', 'browser'],
  },
})
