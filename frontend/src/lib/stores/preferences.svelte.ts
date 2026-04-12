import { Events } from '@wailsio/runtime'
import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'
import { getLogger } from '$lib/logger'

const log = getLogger('preferences')

export interface ResolvedPrefs {
  theme: string
  accentColor: string
  fontSize: number
  compactRows: boolean
  readOnly: boolean
  terminalWebGL: boolean
  keybindings: Record<string, string> | null
  savedFilters: Record<string, SavedFilter[]> | null
  favoriteNamespaces: string[] | null
}

export interface SavedFilter {
  name: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  search?: string
}

class PreferencesStore {
  prefs = $state<ResolvedPrefs>({
    theme: 'system',
    accentColor: '',
    fontSize: 0,
    compactRows: false,
    readOnly: false,
    terminalWebGL: false,
    keybindings: null,
    savedFilters: null,
    favoriteNamespaces: null,
  })

  private activeContext = ''
  private unsub: (() => void) | undefined

  async load(ctxName: string) {
    this.activeContext = ctxName
    try {
      const resolved = await ConfigService.GetResolvedPrefs(ctxName)
      if (resolved) {
        this.prefs = resolved as ResolvedPrefs
      }
    } catch (e) {
      log.error('Failed to load preferences', { error: String(e) })
    }
  }

  subscribe() {
    this.unsub = Events.On('config:updated', () => {
      this.load(this.activeContext)
    })
  }

  destroy() {
    this.unsub?.()
  }

  getKeybinding(actionID: string): string | undefined {
    return this.prefs.keybindings?.[actionID]
  }

  getSavedFilters(gvr: string): SavedFilter[] {
    return this.prefs.savedFilters?.[gvr] ?? []
  }
}

export const preferencesStore = new PreferencesStore()
