# Phase 5 — Enrichers & Expanded Columns

Implement 11 new or extended Go enrichers and add hidden columns to all builtin descriptors, giving users access to richer resource information when they enable columns via the column menu.

## First Action

Read `internal/resource/enrichers/pod.go` — it's the clearest example of the enricher pattern: `Enrich(obj *unstructured.Unstructured) error`, extract fields with `unstructured.NestedFieldNoCopy`, set display fields with `unstructured.SetNestedField`. Every enricher you build follows this exact shape.

## Context

Phase 1 added `Hidden bool` to the `Column` struct and Namespace columns to all namespaced descriptors. This phase runs in parallel with Phases 2–4 (frontend work) and only depends on Phase 1's Go changes. The goal: enrich builtin resource objects with computed display fields (ports summary, access modes, owner references, etc.) and add hidden columns to descriptors that reference those fields. Users will later enable these columns via the column menu (Phases 3–4).

## Files to Read

- `internal/resource/enrichers/pod.go` — **what to look for**: the `Enrich` method pattern — how it extracts nested fields from unstructured objects and sets computed display strings. This is the template for all new enrichers.
- `internal/resource/enrichers/job.go` — **what to look for**: the `JobEnricher` you'll extend with `statusDisplay`. Note how it computes display fields from conditions.
- `internal/resource/enrichers/daemonset.go` — **what to look for**: the `DaemonSetEnricher` you'll extend with `nodeSelectorDisplay`. Currently minimal.
- `internal/resource/enrichers/node.go` — **what to look for**: the `NodeEnricher` you'll extend with `internalIPDisplay` and `osArchDisplay`. Note the `DrainStateProvider` dependency pattern.
- `internal/resource/enricher.go` — **what to look for**: the `Enricher` interface definition and `EnricherRegistry` with `Register`/`GetAll` methods.
- `internal/resource/builtin.go` — **what to look for**: the `builtinDescriptors` slice (where you'll add new hidden columns) and `RegisterBuiltin()` function (where you'll register new enrichers).

## Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §Phase 2 tables listing every new column per GVR, enricher requirements, and the enricher summary table. This is the authoritative reference for field names, expressions, and which enrichers are new vs extended.
- `PHASES.md` — Phase 5 section for the complete test list and acceptance criteria.

## What Exists

- `Column` struct with `Hidden bool` field (Phase 1)
- Existing enrichers: `PodEnricher`, `DeploymentEnricher`, `StatefulSetEnricher`, `DaemonSetEnricher`, `JobEnricher`, `NodeEnricher`, `StorageClassEnricher`, `CRDEnricher`
- `Enricher` interface: `Enrich(obj *unstructured.Unstructured) error`
- `EnricherRegistry` with slice-based storage: `Register(gvr, enricher)`, `GetAll(gvr) []Enricher`
- `RegisterBuiltin()` in `builtin.go` that registers all existing enrichers
- All builtin descriptors with their current column sets (some now include hidden Namespace columns from Phase 1)

## Deliverables

1. **Extend `DaemonSetEnricher`** — add `status.nodeSelectorDisplay`: flatten `spec.nodeSelector` map to `"key=val,key2=val2"` string
2. **Extend `JobEnricher`** — add `status.statusDisplay`: compute `"Complete"`, `"Failed"`, or `"Running"` from `status.conditions`
3. **Extend `NodeEnricher`** — add `status.internalIPDisplay` (find `status.addresses[]` where `type == "InternalIP"`) and `status.osArchDisplay` (combine `status.nodeInfo.operatingSystem` + `"/"` + `status.nodeInfo.architecture`)
4. **New `ReplicaSetEnricher`** — `status.ownerDisplay`: first `metadata.ownerReferences[].name`, or `"<none>"`
5. **New `CronJobEnricher`** — `status.activeCount`: `len(status.active)` as int64
6. **New `ServiceEnricher`** — `status.externalIPDisplay` (flatten `status.loadBalancer.ingress[].ip` or `spec.externalIPs[]`) and `status.portsDisplay` (format `spec.ports[]` as `"80/TCP, 443/TCP"`, include nodePort if non-zero: `"80:30080/TCP"`)
7. **New `IngressEnricher`** — `status.hostsDisplay` (comma-separated `spec.rules[].host`) and `status.defaultBackendDisplay` (format `spec.defaultBackend.service.name:port`)
8. **New `ConfigMapEnricher`** — `status.dataKeysCount`: `len(data)` as int64
9. **New `SecretEnricher`** — `status.dataKeysCount`: `len(data)` as int64
10. **New `PVEnricher`** — `status.accessModesDisplay` (abbreviate: RWO, ROX, RWX, RWOP) and `status.claimDisplay` (format `claimRef.namespace/claimRef.name`)
11. **New `PVCEnricher`** — `status.accessModesDisplay` (same abbreviation logic as PV)
12. **New `ServiceAccountEnricher`** — `status.secretsCount`: `len(secrets)` as int64
13. **New `RoleEnricher`** — `status.rulesCount`: `len(rules)` as int64 (used for both roles and clusterroles)
14. **New `BindingEnricher`** — `status.roleRefDisplay` (format `roleRef.kind/roleRef.name`) and `status.subjectsCount` (`len(subjects)` as int64). Used for both rolebindings and clusterrolebindings.
15. All new enrichers registered in `RegisterBuiltin()` with correct GVR keys
16. New hidden columns added to every affected descriptor per the spec tables (all with `Hidden: true`)

## Tests

- **Go unit test (per enricher)** — each test creates an `unstructured.Unstructured` with the relevant fields, calls `Enrich()`, and asserts the computed display field value:
  - `TestDaemonSetEnricher_NodeSelectorDisplay` — `{disktype: ssd, zone: us-east}` → `"disktype=ssd,zone=us-east"` (sorted keys)
  - `TestReplicaSetEnricher_OwnerDisplay` — one ownerReference → owner name
  - `TestReplicaSetEnricher_NoOwner` — no ownerReferences → `"<none>"`
  - `TestJobEnricher_StatusDisplay` — Complete condition → `"Complete"`
  - `TestCronJobEnricher_ActiveCount` — 2 active items → int64(2)
  - `TestServiceEnricher_PortsDisplay` — two ports → `"80/TCP, 443/TCP"`
  - `TestServiceEnricher_ExternalIPDisplay` — loadBalancer with one ingress IP
  - `TestIngressEnricher_HostsDisplay` — two rules → `"foo.com, bar.com"`
  - `TestIngressEnricher_DefaultBackendDisplay` — service backend → `"my-service:8080"`
  - `TestConfigMapEnricher_DataKeysCount` — 3 data keys → int64(3)
  - `TestSecretEnricher_DataKeysCount` — 2 data keys → int64(2)
  - `TestPVEnricher_AccessModesDisplay` — `[ReadWriteOnce, ReadOnlyMany]` → `"RWO,ROX"`
  - `TestPVEnricher_ClaimDisplay` — claimRef → `"default/my-pvc"`
  - `TestPVCEnricher_AccessModesDisplay` — same abbreviation as PV
  - `TestNodeEnricher_InternalIPDisplay` — addresses array with InternalIP → correct IP
  - `TestNodeEnricher_OsArchDisplay` — → `"linux/amd64"`
  - `TestServiceAccountEnricher_SecretsCount` — 2 secrets → int64(2)
  - `TestRoleEnricher_RulesCount` — 3 rules → int64(3)
  - `TestBindingEnricher_RoleRefDisplay` — → `"ClusterRole/admin"`
  - `TestBindingEnricher_SubjectsCount` — 2 subjects → int64(2)

- **Go integration**
  - `go test ./internal/resource/... -v` passes (all enrichers, descriptors, registry)

## Acceptance Criteria

- [ ] All 11 new/extended enrichers implemented in `internal/resource/enrichers/`
- [ ] Each enricher has at least one unit test covering the happy path
- [ ] Edge cases tested: empty/nil inputs return sensible defaults (empty string, 0)
- [ ] All new columns added to builtin descriptors with `Hidden: true`
- [ ] Enrichers registered in `RegisterBuiltin()` with correct GVR strings
- [ ] `go test ./internal/resource/... -v` passes
- [ ] `go test ./internal/resource/enrichers/ -v` passes
- [ ] No existing enricher tests broken

## Definition of Done

Running `go test ./internal/resource/enrichers/ -v` shows all new and existing enricher tests passing. `go test ./internal/resource/ -v` passes (descriptors validate, registry works). Inspecting `builtin.go` shows new hidden columns on all affected descriptors and new enricher registrations in `RegisterBuiltin()`. A quick manual check: enriching a pod object produces expected `status.readyDisplay`, and enriching a service object produces expected `status.portsDisplay`.

## Known Gotchas

- **Enrichers set fields under `status.*` even when source is `spec.*`.** This is an established codebase pattern. Don't fight it — the CEL expressions in column definitions reference the enriched paths (e.g. `status.portsDisplay` even though the data comes from `spec.ports`). The enricher writes to a path the column expects.

- **`unstructured.SetNestedField` requires `int64` for numeric values, not `int`.** If you set `len(data)` directly, it's an `int` which won't round-trip through JSON correctly. Cast to `int64` explicitly: `unstructured.SetNestedField(obj.Object, int64(len(data)), "status", "dataKeysCount")`.

- **Access mode abbreviation map must be complete.** `ReadWriteOnce→RWO`, `ReadOnlyMany→ROX`, `ReadWriteMany→RWX`, `ReadWriteOncePod→RWOP`. If an unknown mode appears, pass it through unabbreviated rather than silently dropping it.

- **Service ports format must match kubectl.** Format: `port/protocol`. Include nodePort if non-zero: `port:nodePort/protocol` (e.g. `"80:30080/TCP"`). Don't include targetPort — kubectl doesn't show it in the column view.

- **DaemonSet nodeSelector keys should be sorted.** Map iteration order in Go is random. Sort the keys alphabetically before joining to produce deterministic output for tests.

- **ConfigMap/Secret enricher: use `len(data)` only, not `len(data) + len(binaryData)`.** This matches kubectl behavior. The `data` field is the top-level `data` map on the object, not `spec.data`.

- **RoleEnricher is used for BOTH `roles` AND `clusterroles`.** Register it for both GVRs in `RegisterBuiltin()`. Same for `BindingEnricher` — register for both `rolebindings` and `clusterrolebindings`.
