import {streamingStore} from "$lib/stores/streaming.svelte.js";

export function termlog(msg: string): void {
  const cfg = streamingStore.config;
  if (!cfg) {
    return;
  }
  fetch(`http://127.0.0.1:${cfg.port}/${cfg.token}/log`, {
    method: "POST",
    body: msg,
  }).catch(() => {
    /* best-effort */
  });
}
