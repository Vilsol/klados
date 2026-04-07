# Klados — v2 Sprint Plan

Derived from FEATURES.md v2 items, ordered by value/effort ratio and grouped by implementation cohesion.

---

## Sprint 1 — Workload Rollout Operations

**Rationale:** Tier 1 operational value. Rollout history infrastructure (ReplicaSet revision annotations) is built once and reused across all workload types. CronJob/Job operations fit naturally in the same sprint.

### Deployments
- [ ] Rollout history — list revisions with ReplicaSet annotations
- [ ] Rollback to specific revision
- [ ] Pause / resume rollout

### StatefulSets
- [ ] Scale replicas
- [ ] Restart rollout
- [ ] Rollout history

### DaemonSets
- [ ] Rollout history and rollback

### Jobs
- [ ] Delete job with owned pods (cascade)
- [ ] Delete job without owned pods

### CronJobs
- [ ] Trigger manual run (create Job from CronJob spec)
- [ ] Suspend / resume

### Cluster — Node Operations *(no other group, attach here)*
- [ ] Cordon node
- [ ] Uncordon node
- [ ] Drain node (with progress/event view — drain is async)

---

## Sprint 2 — RBAC Section

**Rationale:** Best group cohesion on the board. All items are list+detail viewers with no mutations required. One new sidebar section, lowest complexity-per-feature ratio.

### ServiceAccounts
- [ ] List service accounts
- [ ] Detail view — secrets, image pull secrets, automount token
- [ ] Associated roles/bindings

### Roles / ClusterRoles
- [ ] List roles
- [ ] Detail view — rules (apiGroups, resources, verbs)
- [ ] Rule table with expandable details

### RoleBindings / ClusterRoleBindings
- [ ] List bindings
- [ ] Detail view — subjects, role reference
- [ ] Linked subjects and roles

---

## Sprint 3 — YAML Editor Completions

**Rationale:** Diff view (tier 1 safety feature) anchors a natural group. All CodeMirror 6 extensions in the same editor setup — one focused pass.

- [ ] Diff view before applying changes
- [ ] Schema validation (real-time, against k8s OpenAPI schema)
- [ ] Format / prettify
- [ ] Code folding

---

## Sprint 4 — Storage Completions

**Rationale:** Fills out the existing Storage view with no new navigation needed. Bounded scope, slots into already-built infrastructure.

### StorageClasses
- [ ] List with provisioner, reclaim policy, volume binding mode
- [ ] Default class indicator
- [ ] Parameters display

### CSI Drivers
- [ ] List installed CSI drivers
- [ ] Capabilities (volume snapshot, expansion, etc.)

### PersistentVolumeClaims
- [ ] Expand PVC (if storage class allows)
- [ ] Delete PVC

---

## Sprint 5 — CRD Management

**Rationale:** Entirely self-contained section, no dependencies on prior sprints. Larger lift but clean scope.

- [ ] List all CRDs in cluster
- [ ] CRD detail — group, versions, scope, schema
- [ ] OpenAPI schema viewer for CRD spec
- [ ] List instances of any CRD
- [ ] View / edit / delete CRD instances
- [ ] Auto-discover and render custom resources in sidebar

---

## Sprint 6 — Helm Integration

**Rationale:** Biggest lift, deferred last. Self-contained — uses the Helm Go SDK directly with no overlap with other sprints.

- [ ] List Helm releases across namespaces
- [ ] Release detail — chart, version, status, values
- [ ] Release history / revisions
- [ ] Rollback to previous revision
- [ ] View computed values (user + default merged)
- [ ] View release notes
- [ ] View manifest (rendered templates)
- [ ] Uninstall release
