// Type exports for plugin authors
export type { PluginManifest, Permissions, ResourcePermission, Extensions, DetailTab, Command, SidebarEntry } from '../../frontend/src/lib/plugins/types/manifest.js'
export type { PluginContext, K8SContext, LogsContext, ExecContext, StorageContext } from '../../frontend/src/lib/plugins/types/context.js'

import type { UserConfig } from 'vite'

/**
 * Vite config helper for klados plugin authors.
 * Marks svelte and @klados/plugin-sdk as external so they're
 * not bundled into the plugin — the host app provides them at runtime.
 */
export function defineKladosPlugin(config: UserConfig = {}): UserConfig {
  return {
    ...config,
    build: {
      ...config.build,
      lib: {
        entry: 'src/index.ts',
        formats: ['es'],
        ...(config.build?.lib as object | undefined ?? {}),
      },
      rollupOptions: {
        ...config.build?.rollupOptions,
        external: [
          'svelte',
          'svelte/internal',
          'svelte/internal/client',
          '@klados/plugin-sdk',
          ...(Array.isArray(config.build?.rollupOptions?.external) ? config.build.rollupOptions.external : []),
        ],
      },
    },
  }
}
