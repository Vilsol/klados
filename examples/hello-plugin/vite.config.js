import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  build: {
    lib: {
      entry: 'src/HelloTab.svelte',
      fileName: 'HelloTab',
      formats: ['es'],
    },
    outDir: 'ui',
    emptyOutDir: false,
    minify: false,
    sourcemap: true,
    rollupOptions: {
      external: ['svelte', 'svelte/internal', 'svelte/internal/client'],
    },
  },
})
