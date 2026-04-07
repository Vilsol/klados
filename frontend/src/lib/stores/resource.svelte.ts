import { Events } from '@wailsio/runtime'
import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'

interface WatchEvent {
  type: 'ADDED' | 'MODIFIED' | 'DELETED'
  object: Record<string, any>
}

function resourceKey(obj: Record<string, any>): string {
  return `${obj.metadata?.namespace ?? ''}/${obj.metadata?.name ?? ''}`
}

class ResourceStore {
  items = $state<Record<string, any>[]>([])
  loading = $state(false)
  error = $state<string | null>(null)

  private contextName = ''
  private gvr = ''
  private namespace = ''
  private eventName = ''
  private unsub: (() => void) | null = null
  private generation = 0

  async start(contextName: string, gvr: string, namespace: string) {
    this.stop()
    const gen = this.generation  // captured after stop() bumped it

    this.contextName = contextName
    this.gvr = gvr
    this.namespace = namespace
    this.eventName = `watch:${contextName}:${gvr}:${namespace}`
    this.loading = true
    this.error = null

    // Events.On returns an unsubscribe fn; callback receives WailsEvent { name, data }
    this.unsub = Events.On(this.eventName, (wailsEvent: any) => {
      this.handleEvent(wailsEvent.data as WatchEvent)
    })

    try {
      const list = await ResourceService.ListResources(contextName, gvr, namespace)
      if (gen !== this.generation) return  // superseded by a newer start/stop

      const map = new Map<string, Record<string, any>>()
      for (const obj of list ?? []) {
        map.set(resourceKey(obj), obj)
      }
      this.items = Array.from(map.values())

      await ResourceService.StartWatch(contextName, gvr, namespace)
    } catch (e: any) {
      if (gen !== this.generation) return
      this.error = e?.message ?? String(e)
    } finally {
      if (gen === this.generation) this.loading = false
    }
  }

  stop() {
    this.generation++
    if (this.unsub) {
      this.unsub()
      this.unsub = null
    }
    if (this.contextName && this.gvr) {
      ResourceService.StopWatch(this.contextName, this.gvr, this.namespace).catch(() => {})
    }
    this.items = []
    this.loading = false
    this.error = null
  }

  private handleEvent(event: WatchEvent) {
    if (!event?.object) return
    const obj = event.object
    const key = resourceKey(obj)

    if (event.type === 'DELETED') {
      this.items = this.items.filter((i) => resourceKey(i) !== key)
    } else {
      const idx = this.items.findIndex((i) => resourceKey(i) === key)
      if (idx >= 0) {
        const next = [...this.items]
        next[idx] = obj
        this.items = next
      } else {
        this.items = [...this.items, obj]
      }
    }
  }
}

export function createResourceStore() {
  return new ResourceStore()
}
