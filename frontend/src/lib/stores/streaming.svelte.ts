import {Events} from "@wailsio/runtime";
import {GetStreamingConfig} from "../../../bindings/github.com/Vilsol/klados/internal/services/appservice.js";

export interface StreamingConfig {
  port: number;
  token: string;
}

class StreamingStore {
  config = $state<StreamingConfig | null>(null);

  constructor() {
    if (typeof window !== "undefined") {
      Events.On("streaming:ready", (wailsEvent: unknown) => {
        this.config = ((wailsEvent as {data?: unknown})?.data ?? wailsEvent) as StreamingConfig;
      });

      // The event may have already fired before this listener was registered.
      // Fetch the current config directly as a fallback.
      GetStreamingConfig()
        .then((cfg: StreamingConfig) => {
          if (cfg?.port && !this.config) {
            this.config = cfg;
          }
        })
        .catch(() => {
          /* not yet available */
        });
    }
  }
}

export const streamingStore = new StreamingStore();
