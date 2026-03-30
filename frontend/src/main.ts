import { mount } from 'svelte'
import App from './App.svelte'
import './app.css'
import { streamingStore } from '$lib/stores/streaming.svelte.js'

// Patch console so all output is forwarded to the terminal via the streaming server.
// Logs before the streaming server is ready are buffered and flushed on connect.
const queue: string[] = []

function send(msg: string) {
  const cfg = streamingStore.config
  if (!cfg) { queue.push(msg); return }
  while (queue.length) {
    const m = queue.shift()!
    fetch(`http://127.0.0.1:${cfg.port}/${cfg.token}/log`, { method: 'POST', body: m }).catch(() => {})
  }
  fetch(`http://127.0.0.1:${cfg.port}/${cfg.token}/log`, { method: 'POST', body: msg }).catch(() => {})
}

function fmt(prefix: string, args: unknown[]) {
  return prefix + args.map(a => typeof a === 'object' ? JSON.stringify(a) : String(a)).join(' ')
}

const _log = console.log.bind(console)
const _warn = console.warn.bind(console)
const _error = console.error.bind(console)
const _debug = console.debug.bind(console)

console.log   = (...a) => { _log(...a);   send(fmt('[LOG] ', a)) }
console.warn  = (...a) => { _warn(...a);  send(fmt('[WRN] ', a)) }
console.error = (...a) => { _error(...a); send(fmt('[ERR] ', a)) }
console.debug = (...a) => { _debug(...a); send(fmt('[DBG] ', a)) }

const app = mount(App, { target: document.getElementById('app')! })

export default app
