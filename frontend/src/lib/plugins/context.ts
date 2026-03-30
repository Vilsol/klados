import { Events } from '@wailsio/runtime'
import type { PluginManifest } from './types/manifest.js'
import type { PluginContext } from './types/context.js'
import { assertGVRPermission } from './permissions.js'
import * as LogService from '../../../bindings/github.com/Vilsol/klados/internal/services/logservice.js'
import * as ExecService from '../../../bindings/github.com/Vilsol/klados/internal/services/execservice.js'
import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'
import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'

export interface HostServices {
  clusterName: string
  clusterVersion: string
  namespace: string
  listResources(gvr: string, ns?: string): Promise<unknown[]>
  getResource(gvr: string, ns: string, name: string): Promise<unknown>
}

export function createPluginContext(manifest: PluginManifest, host: HostServices): Readonly<PluginContext> {
  const ctx: Record<string, unknown> = {
    cluster: Object.freeze({ name: host.clusterName, version: host.clusterVersion }),
    namespace: host.namespace,
  }

  if (manifest.permissions?.resources?.length) {
    ctx.k8s = Object.freeze({
      list: (gvr: string, ns?: string) => {
        assertGVRPermission(manifest, gvr, 'list')
        return host.listResources(gvr, ns)
      },
      get: (gvr: string, ns: string, name: string) => {
        assertGVRPermission(manifest, gvr, 'get')
        return host.getResource(gvr, ns, name)
      },
      watch: (gvr: string, ns: string, callback: (event: unknown) => void) => {
        assertGVRPermission(manifest, gvr, 'watch')
        const clusterName = host.clusterName
        const watchNs = ns ?? ''
        ResourceService.StartWatch(clusterName, gvr, watchNs)
        const eventName = `watch:${clusterName}:${gvr}:${watchNs}`
        const unsub = Events.On(eventName, (wailsEvent: any) => callback(wailsEvent.data))
        return () => {
          unsub()
          ResourceService.StopWatch(clusterName, gvr, watchNs)
        }
      },
    })
  }

  if (manifest.permissions?.events) {
    ctx.events = Object.freeze({
      subscribe: (eventName: string, cb: (payload: unknown) => void) => {
        return Events.On('plugin:event', (wailsEvent: any) => {
          const data = wailsEvent.data as { eventName: string; payload: unknown } | undefined
          if (data?.eventName === eventName) {
            cb(data.payload)
          }
        })
      },
    })
  }

  if (manifest.permissions?.logs) {
    ctx.logs = Object.freeze({
      stream: (pod: string, ns: string, container: string, opts?: { follow?: boolean; previous?: boolean; tailLines?: number }) =>
        LogService.StartLogStream(host.clusterName, ns, pod, {
          container,
          follow: opts?.follow ?? false,
          previous: opts?.previous ?? false,
          tailLines: opts?.tailLines != null ? opts.tailLines : null,
          timestamps: false,
        } as any),
      stop: (streamID: string) => LogService.StopLogStream(streamID),
    })
  }

  if (manifest.permissions?.exec) {
    ctx.exec = Object.freeze({
      open: (pod: string, ns: string, container: string, shell?: string) =>
        ExecService.OpenExecSession(host.clusterName, ns, pod, container, shell ?? '/bin/sh'),
      close: (sessionID: string) => ExecService.CloseExecSession(sessionID),
    })
  }

  if (manifest.permissions?.storage) {
    const pluginName = manifest.name
    ctx.storage = Object.freeze({
      get: (key: string) => PluginService.GetPluginStorageKey(pluginName, key),
      set: (key: string, value: string) => PluginService.SetPluginStorageKey(pluginName, key, value),
      delete: (key: string) => PluginService.DeletePluginStorageKey(pluginName, key),
    })
  }

  return Object.freeze(ctx as unknown as PluginContext)
}
