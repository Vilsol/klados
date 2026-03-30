import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  build: {
    lib: {
      entry: {
        NodeAnnotation: 'src/NodeAnnotation.svelte',
        NodeTaintBadge: 'src/NodeTaintBadge.svelte',
        NodeContextItem: 'src/NodeContextItem.svelte',
        NodeHeaderWidget: 'src/NodeHeaderWidget.svelte',
        NodeStatusWidget: 'src/NodeStatusWidget.svelte',
      },
      formats: ['es'],
    },
    outDir: '../ui',
    emptyOutDir: false,
    minify: false,
    sourcemap: true,
    rollupOptions: {
      external: ['svelte', 'svelte/internal', 'svelte/internal/client'],
    },
  },
})
