# Klados — Feature Plan

> The Pod is a Lie

Full-scale feature inventory for a Kubernetes desktop IDE built on Go + Wails 3 + Svelte.

**Priority tags:**
- **`MVP`** — required for first usable release
- **`v2`** — second phase
- **`Plugin`** — best implemented as a plugin (proves the plugin system)
- **`Future`** — nice to have, low priority

---

## 1. Cluster Management

- [x] `MVP` Multi-cluster support — add/remove/switch clusters
- [x] `MVP` Kubeconfig auto-detection (~/.kube/config, KUBECONFIG env)
- [x] `MVP` Kubeconfig manual import (file picker, paste)
- [ ] `Future` In-cluster context detection (when running inside a pod)
- [x] `MVP` Cluster connection status indicators (connected, degraded, unreachable)
- [x] `v2` Cluster health overview (API server health, component statuses)
- [x] `MVP` Cluster metadata display (version, platform, provider detection)
- [x] `MVP` Namespace management — list, switch active, create, delete
- [x] `v2` Namespace favorites / pinning
- [x] `MVP` Multi-namespace view (all namespaces simultaneously)
- [ ] `v2` Cluster resource usage summary (allocatable vs requested vs used)
- [x] `MVP` Node overview — list, status, conditions, taints, labels
- [x] `MVP` Node resource utilization (CPU, memory, pods, ephemeral storage)
- [x] `v2` Node cordon / uncordon / drain actions
- [ ] `v2` Cluster API version / feature gate discovery
- [x] `v2` MutatingWebhookConfigurations — list, detail with rule summary, failure policy indicators
- [x] `v2` ValidatingWebhookConfigurations — list, detail with rule summary, failure policy indicators
- [x] `v2` PriorityClasses — list with value, global default, preemption policy
- [x] `v2` RuntimeClasses — list with handler
- [ ] `Future` Workspace grouping — organize clusters into named workspaces
- [ ] `Future` Hotbar — pinnable quick-access bar for frequently used clusters/resources
- [ ] `Future` Catalog — unified entity browser aggregating clusters, services, and custom entities
- [x] `v2` Read-only mode — lockdown flag for safe browsing (e.g. prod clusters)

## 2. Workloads

### Pods
- [x] `MVP` List pods with status, restarts, age, node, IP
- [x] `MVP` Pod detail view — containers, init containers, ephemeral containers
- [x] `MVP` Container status breakdown (waiting, running, terminated + reasons)
- [x] `MVP` Pod conditions (Ready, Initialized, PodScheduled, ContainersReady)
- [x] `v2` Pod resource requests/limits vs actual usage
- [x] `MVP` Pod environment variables (resolved, including configmap/secret refs)
- [x] `MVP` Pod volume mounts visualization
- [x] `MVP` Pod annotations and labels (view/edit)
- [x] `MVP` Pod events timeline
- [x] `MVP` Pod delete / force delete
- [x] `MVP` Pod YAML view / edit / apply
- [x] `MVP` Multi-container pod log switching

### Deployments
- [x] `MVP` List deployments with replicas, status, age
- [x] `MVP` Deployment detail — strategy, selectors, conditions
- [x] `MVP` Scale replicas (slider or input)
- [x] `MVP` Restart (rollout restart)
- [x] `v2` Rollout history and rollback to specific revision
- [x] `v2` Pause / resume rollout
- [x] `MVP` Edit deployment YAML / apply

### StatefulSets
- [x] `MVP` List with replicas, status
- [x] `MVP` Detail view — update strategy, volume claim templates
- [x] `v2` Scale replicas
- [x] `v2` Restart rollout
- [x] `v2` Rollout history

### DaemonSets
- [x] `MVP` List with desired/current/ready counts
- [x] `MVP` Detail view — update strategy, node selector
- [x] `v2` Rollout history and rollback

### ReplicaSets
- [x] `MVP` List with desired/current/ready
- [x] `MVP` Detail view — owned pods
- [ ] `v2` Scale (rarely used directly, but available)

### Jobs
- [x] `MVP` List with completions, duration, status
- [x] `MVP` Detail view — parallelism, backoff limit, active deadline
- [x] `MVP` View owned pods
- [x] `v2` Delete job (with/without owned pods)

### CronJobs
- [x] `MVP` List with schedule, last run, active, suspended
- [x] `MVP` Detail view — concurrency policy, history limits
- [x] `v2` Trigger manual run
- [x] `v2` Suspend / resume
- [x] `MVP` View job history

## 3. Networking

### Services
- [x] `MVP` List services with type, cluster IP, external IP, ports
- [x] `MVP` Detail view — selectors, endpoints, session affinity
- [x] `MVP` Port-forward action (with local port selection)
- [x] `MVP` Endpoint resolution — show backing pods

### Ingresses
- [x] `MVP` List with hosts, paths, backends
- [x] `MVP` Detail view — TLS, rules, annotations
- [x] `MVP` Link to open in browser (for accessible hosts)

### IngressClasses
- [x] `v2` List available ingress classes
- [x] `v2` Default class indicator

### NetworkPolicies
- [x] `v2` List policies
- [x] `v2` Detail view — ingress/egress rules, pod selectors
- [x] `v2` Visual rule summary (which pods can talk to which)

### Endpoints / EndpointSlices
- [x] `v2` List and detail view
- [x] `v2` Show associated service and target pods

### Port Forwarding Manager
- [x] `MVP` Active port-forwards list
- [x] `MVP` Start/stop port-forwards
- [x] `v2` Port-forward persistence (saved forwards survive app restarts, keyed by context)
- [x] `v2` Port-forward management page (`/c/:ctx/port-forwards`) with enable/disable/remove actions
- [x] `v2` Auto-reconnect on disconnect (enabled forwards auto-reconnect on cluster connect)
- [x] `MVP` Status indicator (active, failed, reconnecting)

## 4. Configuration

### ConfigMaps
- [x] `MVP` List configmaps
- [x] `MVP` Detail view — data keys with syntax-highlighted values
- [x] `MVP` Edit individual keys or full YAML
- [x] `v2` Create new configmap
- [x] `MVP` Delete

### Secrets
- [x] `MVP` List secrets with type
- [x] `MVP` Detail view — data keys (base64 decoded toggle)
- [x] `MVP` Show/hide secret values
- [x] `MVP` Edit individual keys or full YAML
- [x] `v2` Create new secret
- [x] `MVP` Copy decoded value to clipboard
- [x] `MVP` Delete

### Resource Quotas
- [x] `v2` List quotas per namespace
- [x] `v2` Usage vs hard limits display

### Limit Ranges
- [x] `v2` List and detail view
- [x] `v2` Default/max/min per resource type

### HPAs (Horizontal Pod Autoscalers)
- [x] `v2` List with min/max/current replicas, target metrics
- [x] `v2` Detail view — scaling behavior, metrics status
- [ ] `v2` Edit min/max replicas
- [x] `v2` Current vs target metric values

### VPAs (Vertical Pod Autoscalers)
- [ ] `Future` List and detail view
- [ ] `Future` Recommendations display

### PDBs (Pod Disruption Budgets)
- [x] `v2` List with allowed disruptions
- [x] `v2` Detail view — min available / max unavailable

### Leases
- [x] `v2` List leases with holder, duration, renew time
- [x] `v2` Detail view — leader election debugging

## 5. Storage

### PersistentVolumes (PV)
- [x] `MVP` List PVs with capacity, access modes, reclaim policy, status
- [x] `MVP` Detail view — source (NFS, hostPath, CSI, etc.), mount options
- [x] `MVP` Bound PVC association

### PersistentVolumeClaims (PVC)
- [x] `MVP` List PVCs with status, capacity, storage class, bound PV
- [x] `MVP` Detail view — access modes, volume mode
- [x] `v2` Expand PVC (if storage class allows)
- [x] `v2` Delete PVC

### StorageClasses
- [x] `v2` List with provisioner, reclaim policy, volume binding mode
- [x] `v2` Default class indicator
- [x] `v2` Parameters display

### CSI Drivers
- [x] `v2` List installed CSI drivers
- [x] `v2` Capabilities (volume snapshot, expansion, etc.)

## 6. RBAC & Security

### ServiceAccounts
- [x] `v2` List service accounts
- [x] `v2` Detail view — secrets, image pull secrets, automount token
- [x] `v2` Associated roles/bindings

### Roles / ClusterRoles
- [x] `v2` List roles
- [x] `v2` Detail view — rules (apiGroups, resources, verbs)
- [x] `v2` Rule table with expandable details

### RoleBindings / ClusterRoleBindings
- [x] `v2` List bindings
- [x] `v2` Detail view — subjects, role reference
- [x] `v2` Linked subjects and roles

### RBAC Visualization
- [ ] `Future` Who-can query — "who can GET pods in namespace X?"
- [ ] `Future` Subject access review — "what can service account Y do?"
- [ ] `Future` Access matrix view
- [ ] `Future` RBAC-aware UI adaptation — hide/disable actions the user lacks permissions for

## 7. Custom Resources

### CRD Management
- [x] `v2` List all CRDs in cluster
- [x] `v2` CRD detail — group, versions, scope, schema
- [x] `v2` OpenAPI schema viewer for CRD spec
- [x] `v2` List instances of any CRD
- [x] `v2` View/edit/delete CRD instances
- [x] `v2` Auto-discover and render custom resources in sidebar

### Common CRD Integrations
- [ ] `Plugin` Prometheus: ServiceMonitor, PodMonitor, PrometheusRule
- [ ] `Plugin` Cert-Manager: Certificate, Issuer, ClusterIssuer
- [ ] `Plugin` Istio: VirtualService, DestinationRule, Gateway
- [ ] `Plugin` ArgoCD: Application, AppProject
- [ ] `Plugin` Flux: Kustomization, HelmRelease, GitRepository
- [ ] `Plugin` Knative: Service, Revision, Route
- [ ] `Plugin` Crossplane: Composite resources
- [ ] `Plugin` Gateway API: HTTPRoute, Gateway, GatewayClass

## 8. Observability & Debugging

### Logs
- [x] `MVP` Real-time log streaming (follow mode)
- [x] `MVP` Historical log retrieval with line limits
- [x] `MVP` Multi-container log selection
- [x] `MVP` Init container logs
- [x] `MVP` Previous container logs (after restart)
- [x] `MVP` Log search / filter (regex)
- [x] `MVP` Log highlighting (error, warn, info levels)
- [x] `MVP` Timestamp toggle
- [x] `MVP` Log download / export
- [x] `v2` Multi-pod log aggregation (tail logs from all pods in a deployment)
- [x] `MVP` Log wrapping toggle
- [x] `v2` Font size control

### Terminal / Shell
- [x] `MVP` Exec into running containers (interactive shell)
- [x] `MVP` Container selection for multi-container pods
- [x] `MVP` Shell selection (bash, sh, zsh, etc.)
- [x] `MVP` Multiple concurrent terminal sessions
- [x] `MVP` Terminal tabs
- [x] `MVP` Copy/paste support
- [x] `MVP` Terminal resize handling

### Events
- [x] `MVP` Cluster-wide event stream
- [x] `MVP` Namespace-scoped events
- [x] `MVP` Resource-specific events (on detail pages)
- [x] `MVP` Event filtering (type, reason, source)
- [x] `MVP` Warning event highlighting

### Resource Metrics
- [x] `v2` Node CPU/memory graphs (requires metrics-server)
- [x] `v2` Pod CPU/memory graphs
- [x] `v2` Container-level metrics
- [x] `v2` Namespace resource usage aggregation
- [x] `v2` Historical metric trends (if metrics available)
- [x] `v2` Prometheus auto-detection — discover in-cluster Prometheus and wire up metrics automatically
- [x] `v2` Custom Prometheus endpoint configuration (supports Thanos, Mimir, VictoriaMetrics)
- [x] `v2` Resource requests/limits overlay on CPU/memory graphs
- [x] `v2` OOMKill / CPU throttling / warning event annotations on graphs
- [x] `v2` Multi-container overlay charts with legend toggle
- [x] `v2` Sparkline columns in resource lists (opt-in, batch-queried)
- [x] `v2` Plugin-extensible metric queries via descriptor templates
- [ ] `Future` Alerting rules viewer — show firing Prometheus alerts per resource (Plugin candidate)
- [ ] `Future` GPU metrics — NVIDIA DCGM exporter integration (Plugin candidate)

### Cluster Health
- [ ] `Future` Pulse view — live cluster-wide activity dashboard showing resource churn and recent events
- [ ] `Future` Popeye / Sanitizer — cluster best-practice scanner that flags misconfigurations and scores health
- [ ] `Future` HTTP benchmarking — built-in service benchmarking for load testing

## 9. Helm Integration

- [ ] `v2` List Helm releases across namespaces
- [ ] `v2` Release detail — chart, version, status, values
- [ ] `v2` Release history / revisions
- [ ] `v2` Rollback to previous revision
- [ ] `v2` View computed values (user + default)
- [ ] `v2` View release notes
- [ ] `v2` View manifest (rendered templates)
- [ ] `v2` Uninstall release
- [ ] `Future` Upgrade release with new values
- [ ] `Future` Add/manage Helm repositories
- [ ] `Future` Browse available charts
- [ ] `Future` Install chart with values editor

## 10. Developer Experience

### Resource Creation
- [x] `MVP` Create resource from YAML editor
- [x] `MVP` YAML syntax highlighting and validation
- [ ] `Future` Schema-aware autocomplete for known resource types
- [x] `v2` Template library (common resource templates)
- [x] `v2` Apply from file or clipboard

### Search & Navigation
- [x] `MVP` Global resource search (fuzzy find across all resource types)
- [x] `MVP` Filter by labels
- [x] `v2` Filter by annotations
- [x] `MVP` Sort by any column
- [x] `v2` Column visibility customization
- [x] `v2` Saved filters / views
- [ ] `v2` Keyboard-driven navigation (vim-like bindings optional)
- [x] `MVP` Command palette (Ctrl+K / Cmd+K)
- [x] `MVP` Breadcrumb navigation
- [ ] `v2` Back/forward history

### YAML Editor
- [x] `MVP` Syntax highlighting
- [x] `v2` Schema validation (real-time)
- [x] `v2` Diff view before applying changes
- [x] `MVP` Undo/redo
- [x] `v2` Format/prettify
- [x] `MVP` Line numbers
- [x] `v2` Code folding
- [x] `MVP` Find and replace

### Resource Relationships
- [ ] `v2` Owner reference chain visualization
- [ ] `v2` Deployment → ReplicaSet → Pod hierarchy
- [ ] `v2` Service → Endpoint → Pod mapping
- [ ] `v2` Ingress → Service → Pod flow
- [ ] `v2` PVC → PV binding
- [ ] `v2` ConfigMap/Secret → consuming pods
- [ ] `Future` XRay view — hierarchical tree showing resource ownership chains
- [ ] `Future` Resource relationship graph/map — interactive visual graph of resource connections

## 11. Application Lifecycle

### Multi-Resource Views
- [x] `MVP` "Workloads" view — all deployments, statefulsets, daemonsets, jobs together
- [x] `MVP` "Networking" view — services, ingresses, network policies together
- [x] `MVP` "Storage" view — PVCs, PVs, storage classes together
- [x] `MVP` "Config" view — configmaps, secrets together

### Bulk Operations
- [x] `v2` Multi-select resources
- [x] `v2` Bulk delete
- [x] `v2` Bulk label/annotate
- [x] `v2` Apply YAML with multiple documents (---)
- [ ] `v2` Dir view — apply/watch a local manifest directory against the cluster

### Custom Views
- [ ] `v2` Custom column definitions — user-defined columns per resource type

## 12. UI / UX

### Layout
- [x] `MVP` Sidebar with resource tree / category navigation
- [x] `MVP` Collapsible sidebar
- [x] `MVP` Tab-based multi-resource viewing
- [x] `v2` Split pane support (e.g., list + detail side by side)
- [x] `v2` Resizable panels

### Theming
- [x] `MVP` Dark mode (default)
- [x] `MVP` Light mode
- [x] `MVP` System theme detection
- [x] `v2` Custom accent colors
- [x] `v2` Cluster-specific color coding (to distinguish prod from dev)

### Status & Feedback
- [x] `MVP` Global notification system (toast/snackbar)
- [x] `MVP` Error display with actionable details
- [x] `MVP` Confirmation dialogs for destructive actions

### Accessibility
- [ ] `v2` Keyboard navigation throughout
- [ ] `Future` Screen reader support
- [ ] `Future` High contrast mode option
- [x] `v2` Configurable font size

### Performance
- [x] `MVP` Virtual scrolling for large lists
- [ ] `v2` Incremental resource loading
- [x] `MVP` Watch-based real-time updates (not polling)
- [x] `MVP` Debounced search
- [ ] `v2` Lazy-load detail views

## 13. Plugin System (Wasm + Svelte)

### Plugin Runtime
- [x] `v2` Wasm plugin loading via wazero
- [x] `v2` Plugin manifest format (metadata, permissions, entry points)
- [x] `v2` Plugin lifecycle management (install, enable, disable, uninstall)
- [x] `v2` Plugin sandboxing — scoped API access
- [x] `v2` Plugin-to-host API (k8s client access, UI registration, storage)

### Plugin Capabilities
- [x] `v2` Register new sidebar entries
- [x] `v2` Register new resource views / detail tabs
- [x] `v2` Register custom actions on resources
- [x] `v2` Register status bar widgets
- [x] `v2` Register command palette commands
- [x] `v2` Access cluster data via host-provided k8s API
- [x] `v2` Plugin local storage (preferences, cache)
- [ ] `Future` Inter-plugin communication (optional)

### Plugin UI
- [x] `v2` Svelte component bundles loaded at runtime
- [x] `v2` Plugin UI slots (sidebar, detail tabs, modals, status bar)
- [x] `v2` Host-provided UI component library for consistency
- [x] `v2` Plugin settings page

### Plugin Distribution
- [x] `v2` Local plugin loading (from filesystem)
- [ ] `Future` Plugin registry / marketplace (future)
- [ ] `Future` Plugin versioning and updates
- [ ] `Future` Plugin dependency declaration

## 14. Data Management

### Preferences
- [x] `v2` Per-cluster settings
- [x] `v2` UI preferences (theme, layout, column widths)
- [x] `v2` Keybinding customization
- [x] `v2` Startup behavior (last cluster, specific cluster, chooser)

### Import / Export
- [x] `MVP` Export resource as YAML
- [x] `v2` Export filtered list as YAML/JSON
- [x] `MVP` Copy resource YAML to clipboard
- [x] `MVP` Import YAML from clipboard

## 15. Security & Auth

### Authentication
- [x] `MVP` kubeconfig token auth
- [x] `MVP` Client certificate auth
- [ ] `v2` OIDC / SSO via kubeconfig
- [x] `MVP` Exec-based auth (AWS EKS, GKE, etc.)
- [x] `MVP` Auth token refresh handling

### Local Security
- [x] `MVP` No telemetry / phone-home
- [ ] `v2` Secrets stored via OS keychain (optional)
- [x] `MVP` No cloud dependency — fully offline capable
- [ ] `v2` Configurable kubeconfig file permissions check

## 16. System

### Updates
- [ ] `Future` Update check (opt-in)
- [ ] `Future` In-app update mechanism
- [ ] `Future` Changelog display

### Platform
- [x] `MVP` Linux (primary)
- [x] `v2` macOS support
- [ ] `v2` Windows support
- [ ] `Future` System tray with cluster status
- [ ] `Future` Native OS notifications for watch alerts (optional)
- [x] `Future` Multi-window support
