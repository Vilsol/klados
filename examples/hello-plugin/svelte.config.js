import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  preprocess: vitePreprocess(),
  onwarn(warning, defaultHandler) {
    // Intentionally capture initial prop values at mount time instead of using $effect.
    // This avoids the two-Svelte-instances effect_orphan error in bundled plugins.
    if (warning.code === 'state_referenced_locally') return
    defaultHandler(warning)
  },
}
