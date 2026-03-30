import { Events } from '@wailsio/runtime'
import * as AppService from '../../../bindings/github.com/Vilsol/klados/internal/services/appservice.js'

export interface StreamingConfig {
  port: number
  token: string
}

class StreamingStore {
  config = $state<StreamingConfig | null>(null)

  constructor() {
    if (typeof window !== 'undefined') {
      Events.On('streaming:ready', (wailsEvent: any) => {
        this.config = (wailsEvent.data ?? wailsEvent) as StreamingConfig
      })

      // The event may have already fired before this listener was registered.
      // Fetch the current config directly as a fallback.
      AppService.GetStreamingConfig().then((cfg: StreamingConfig) => {
        if (cfg?.port && !this.config) {
          this.config = cfg
        }
      }).catch(() => {/* not yet available */})
    }
  }
}

export const streamingStore = new StreamingStore()
