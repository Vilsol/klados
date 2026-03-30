import type { PluginManifest } from './types/manifest.js'

// Parses dot-separated GVR ("apps.v1.deployments") into {group, version, resource}.
// Splits from the right since group may contain dots.
function parseGVR(gvr: string): { group: string; version: string; resource: string } {
  const parts = gvr.split('.')
  if (parts.length < 3) {
    throw new Error(`invalid GVR: ${gvr}`)
  }
  const resource = parts[parts.length - 1]
  const version = parts[parts.length - 2]
  const group = parts.slice(0, parts.length - 2).join('.')
  // "core" is the display name for the empty group
  return { group: group === 'core' ? '' : group, version, resource }
}

export function assertGVRPermission(manifest: PluginManifest, gvr: string, verb: string): void {
  const { group, version, resource } = parseGVR(gvr)
  const allowed = manifest.permissions?.resources?.some(
    (p) => p.group === group && p.version === version && p.resource === resource && p.verbs.includes(verb as any)
  )
  if (!allowed) {
    throw new Error(`plugin "${manifest.name}" is not permitted to ${verb} ${gvr}`)
  }
}
