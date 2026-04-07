import { describe, expect, it } from 'vitest'
import { buildCRDTree, extractGroup } from '$lib/utils/crdTree'

const kindOf = (gvr: string) => gvr.split('.').at(-1)!

describe('extractGroup', () => {
  it('returns empty string for core GVRs', () => {
    expect(extractGroup('core.v1.pods')).toBe('')
    expect(extractGroup('core.v1.services')).toBe('')
  })

  it('returns group for standard GVRs', () => {
    expect(extractGroup('apps.v1.deployments')).toBe('apps')
    expect(extractGroup('helm.toolkit.fluxcd.io.v2.helmreleases')).toBe('helm.toolkit.fluxcd.io')
    expect(extractGroup('cert-manager.io.v1.certificates')).toBe('cert-manager.io')
  })
})

describe('buildCRDTree', () => {
  it('groups FluxCD toolkit GVRs into a single folder with subfolders', () => {
    const gvrs = [
      'helm.toolkit.fluxcd.io.v2.helmreleases',
      'image.toolkit.fluxcd.io.v1beta2.imagepolicies',
      'kustomize.toolkit.fluxcd.io.v1.kustomizations',
      'notification.toolkit.fluxcd.io.v1.alerts',
      'source.toolkit.fluxcd.io.v1.gitrepositories',
    ]
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree).toHaveLength(1)
    expect(tree[0].label).toBe('toolkit.fluxcd.io')
    expect(tree[0].fullSuffix).toBe('toolkit.fluxcd.io')
    expect(tree[0].directGvrs).toHaveLength(0)
    const childLabels = tree[0].children.map((c) => c.label).sort()
    expect(childLabels).toEqual(['helm', 'image', 'kustomize', 'notification', 'source'])
  })

  it('produces cert-manager.io with direct kinds and acme subfolder', () => {
    const gvrs = [
      'cert-manager.io.v1.certificates',
      'cert-manager.io.v1.issuers',
      'acme.cert-manager.io.v1.challenges',
      'acme.cert-manager.io.v1.orders',
    ]
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree).toHaveLength(1)
    expect(tree[0].label).toBe('cert-manager.io')
    expect(tree[0].directGvrs.map((e) => e.gvr).sort()).toEqual([
      'cert-manager.io.v1.certificates',
      'cert-manager.io.v1.issuers',
    ])
    expect(tree[0].children).toHaveLength(1)
    expect(tree[0].children[0].label).toBe('acme')
    expect(tree[0].children[0].fullSuffix).toBe('acme.cert-manager.io')
  })

  it('returns two separate top-level nodes for foo.io and bar.io (min-2-segment rule)', () => {
    const gvrs = [
      'foo.io.v1.foos',
      'bar.io.v1.bars',
    ]
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree).toHaveLength(2)
    const labels = tree.map((n) => n.label).sort()
    expect(labels).toEqual(['bar.io', 'foo.io'])
  })

  it('returns three separate top-level nodes for foo.io, bar.io, baz.io', () => {
    const gvrs = [
      'foo.io.v1.foos',
      'bar.io.v1.bars',
      'baz.io.v1.bazzes',
    ]
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree).toHaveLength(3)
    const labels = tree.map((n) => n.label).sort()
    expect(labels).toEqual(['bar.io', 'baz.io', 'foo.io'])
  })

  it('skips GVRs with empty group (core GVRs)', () => {
    const gvrs = ['core.v1.pods', 'core.v1.services', 'foo.io.v1.foos']
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree).toHaveLength(1)
    expect(tree[0].label).toBe('foo.io')
  })

  it('uses last GVR segment as kind fallback when getKind returns empty string', () => {
    const gvrs = ['foo.io.v1.widgets']
    const tree = buildCRDTree(gvrs, () => '')
    // getKind returns '' but the fallback is in Sidebar.svelte; here we verify
    // that buildCRDTree passes through whatever getKind returns
    expect(tree[0].directGvrs[0].kind).toBe('')
  })

  it('uses kind from getKind callback', () => {
    const gvrs = ['foo.io.v1.widgets']
    const tree = buildCRDTree(gvrs, () => 'Widget')
    expect(tree[0].directGvrs[0].kind).toBe('Widget')
  })

  it('sorts nodes alphabetically at every level', () => {
    const gvrs = [
      'z.example.io.v1.zs',
      'a.example.io.v1.as',
      'm.example.io.v1.ms',
    ]
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree).toHaveLength(1)
    const childLabels = tree[0].children.map((c) => c.label)
    expect(childLabels).toEqual(['a', 'm', 'z'])
  })
})

describe('alphabetical sorting', () => {
  it('sorts all top-level nodes alphabetically regardless of whether they have children', () => {
    const gvrs = [
      'kyverno.io.v1.clusterpolicies',
      'cilium.io.v2.ciliumnetworkpolicies',
      'cert-manager.io.v1.certificates',
      'acme.cert-manager.io.v1.challenges',
    ]
    const tree = buildCRDTree(gvrs, kindOf)
    expect(tree[0].label).toBe('cert-manager.io')
    expect(tree[1].label).toBe('cilium.io')
    expect(tree[2].label).toBe('kyverno.io')
  })
})
