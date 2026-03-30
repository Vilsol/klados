import type { UserConfig } from 'vite'

export interface PluginViteConfigOptions {
  /** Entry point for the plugin UI. Defaults to 'src/index.ts'. */
  entry?: string
  /** Additional externals beyond svelte and @klados/plugin-ui. */
  extraExternals?: string[]
}

/**
 * Creates a Vite config suitable for building klados plugin UI bundles.
 *
 * Features:
 * - Builds a single ES module (no code splitting)
 * - Externalises svelte and @klados/plugin-ui (provided by the host)
 * - Enables sourcemaps for easier debugging
 *
 * @example
 * // vite.config.ts
 * import { createPluginViteConfig } from '@klados/plugin-sdk/vite'
 * export default createPluginViteConfig({ entry: 'src/MyTab.svelte' })
 */
export function createPluginViteConfig(options: PluginViteConfigOptions = {}): UserConfig {
  const entry = options.entry ?? 'src/index.ts'
  const external = [
    'svelte',
    'svelte/internal',
    'svelte/internal/client',
    '@klados/plugin-ui',
    ...(options.extraExternals ?? []),
  ]

  return {
    build: {
      lib: {
        entry,
        formats: ['es'],
      },
      outDir: 'ui',
      emptyOutDir: false,
      minify: false,
      sourcemap: true,
      rollupOptions: {
        external,
        output: {
          format: 'es',
          inlineDynamicImports: true,
        },
      },
    },
  }
}
