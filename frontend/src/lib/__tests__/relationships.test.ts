import { describe, it, expect } from 'vitest'
import {
  getControllerRef,
  gvrToApiVersion,
  buildKindGVRMap,
  resolveGVR,
  type APIResource,
} from '$lib/utils/relationships'

describe('getControllerRef', () => {
  it('returns the controller:true ref', () => {
    const obj = {
      metadata: {
        ownerReferences: [
          { apiVersion: 'apps/v1', kind: 'ReplicaSet', name: 'rs-abc', uid: 'uid1', controller: true },
        ],
      },
    }
    expect(getControllerRef(obj)).toEqual({
      apiVersion: 'apps/v1',
      kind: 'ReplicaSet',
      name: 'rs-abc',
      uid: 'uid1',
    })
  })

  it('ignores non-controller refs', () => {
    const obj = {
      metadata: {
        ownerReferences: [
          { apiVersion: 'apps/v1', kind: 'ReplicaSet', name: 'rs-abc', uid: 'uid1', controller: false },
          { apiVersion: 'apps/v1', kind: 'Deployment', name: 'dep', uid: 'uid2', controller: true },
        ],
      },
    }
    const ref = getControllerRef(obj)
    expect(ref?.kind).toBe('Deployment')
  })

  it('returns null when no ownerReferences', () => {
    expect(getControllerRef({ metadata: {} })).toBeNull()
    expect(getControllerRef({})).toBeNull()
    expect(getControllerRef(null)).toBeNull()
  })

  it('returns null when none has controller:true', () => {
    const obj = {
      metadata: {
        ownerReferences: [
          { apiVersion: 'apps/v1', kind: 'ReplicaSet', name: 'rs', uid: 'uid1', controller: false },
        ],
      },
    }
    expect(getControllerRef(obj)).toBeNull()
  })
})

describe('gvrToApiVersion', () => {
  it('converts apps.v1.replicasets to apps/v1', () => {
    expect(gvrToApiVersion('apps.v1.replicasets')).toBe('apps/v1')
  })

  it('converts core.v1.pods to v1', () => {
    expect(gvrToApiVersion('core.v1.pods')).toBe('v1')
  })

  it('converts dotted group networking.k8s.io.v1.ingresses to networking.k8s.io/v1', () => {
    expect(gvrToApiVersion('networking.k8s.io.v1.ingresses')).toBe('networking.k8s.io/v1')
  })
})

describe('buildKindGVRMap', () => {
  it('builds correct map from APIResource array', () => {
    const resources: APIResource[] = [
      { gvr: 'apps.v1.replicasets', kind: 'ReplicaSet', namespaced: true },
      { gvr: 'core.v1.pods', kind: 'Pod', namespaced: true },
      { gvr: 'networking.k8s.io.v1.ingresses', kind: 'Ingress', namespaced: true },
    ]
    const map = buildKindGVRMap(resources)
    expect(map.get('apps/v1:ReplicaSet')).toBe('apps.v1.replicasets')
    expect(map.get('v1:Pod')).toBe('core.v1.pods')
    expect(map.get('networking.k8s.io/v1:Ingress')).toBe('networking.k8s.io.v1.ingresses')
  })
})

describe('resolveGVR', () => {
  it('returns GVR for known kinds', () => {
    const map = new Map([['apps/v1:ReplicaSet', 'apps.v1.replicasets']])
    expect(resolveGVR(map, 'apps/v1', 'ReplicaSet')).toBe('apps.v1.replicasets')
  })

  it('returns undefined for unknown kinds', () => {
    const map = new Map<string, string>()
    expect(resolveGVR(map, 'apps/v1', 'Unknown')).toBeUndefined()
  })
})
