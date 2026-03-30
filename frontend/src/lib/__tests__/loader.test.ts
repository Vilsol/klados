import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock notification store before importing loader
vi.mock('$lib/stores/notification.svelte.js', () => ({
  notificationStore: { error: vi.fn() },
}))

import { loadPluginComponent } from '$lib/plugins/loader.js'
import { notificationStore } from '$lib/stores/notification.svelte.js'

describe('loadPluginComponent', () => {
  beforeEach(() => {
    vi.mocked(notificationStore.error).mockClear()
  })

  it('returns null and calls notificationStore.error on import failure', async () => {
    const result = await loadPluginComponent('bad-plugin', 'ui.js', 'http://127.0.0.1:9999')
    expect(result).toBeNull()
    expect(notificationStore.error).toHaveBeenCalledWith(
      'Plugin "bad-plugin" component failed to load',
      expect.any(String)
    )
  })

  it('returns null on second call for same failing URL (cache miss with null sentinel not stored)', async () => {
    // null is not cached — each failing call retries
    const r1 = await loadPluginComponent('bad-plugin', 'missing.js', 'http://127.0.0.1:9998')
    const r2 = await loadPluginComponent('bad-plugin', 'missing.js', 'http://127.0.0.1:9998')
    expect(r1).toBeNull()
    expect(r2).toBeNull()
  })
})
