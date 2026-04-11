# Overview Panel Improvements

## Summary

Improve the OverviewPanel detail view with more resource fields, better interactivity (click-to-copy, raw value tooltips), a compact conditions layout, a dedicated tolerations section, and structured container cards with accordion sections.

## 1. New Detail Fields (Go Descriptors)

Add new `overviewFields` entries in `internal/resource/builtin.go`. No frontend changes — these are CEL expressions evaluated by the existing rendering pipeline.

### Pods
| Label | Expr | RenderType |
|---|---|---|
| Service Account | `spec.serviceAccountName` | text |
| QoS Class | `status.qosClass` | badge |
| Priority | `spec.priority` | text |
| Restart Policy | `spec.restartPolicy` | badge |
| DNS Policy | `spec.dnsPolicy` | text |

### Deployments
| Label | Expr | RenderType |
|---|---|---|
| Strategy | `spec.strategy.type` | badge |
| Service Account | `spec.template.spec.serviceAccountName` | text |
| Revision | `metadata.annotations['deployment.kubernetes.io/revision']` | text |

### StatefulSets
| Label | Expr | RenderType |
|---|---|---|
| Update Strategy | `spec.updateStrategy.type` | badge |
| Service Account | `spec.template.spec.serviceAccountName` | text |
| Service Name | `spec.serviceName` | text |

### DaemonSets
| Label | Expr | RenderType |
|---|---|---|
| Update Strategy | `spec.updateStrategy.type` | badge |
| Service Account | `spec.template.spec.serviceAccountName` | text |

## 2. Tolerations Section

A dedicated collapsible card below the Details card, visible only when the resource has tolerations.

### Details grid link
A count badge in the Details grid (e.g., "Tolerations: 3"). Clicking it scrolls to the tolerations section and expands it if collapsed.

### Toleration display format
Each toleration renders as a compact one-liner: `key : effect (tolerationSeconds)`. Omit parts that aren't set. An `Exists` operator with no key shows as `* : Exists — effect`.

### Scope
Pods directly, and workloads (Deployments, StatefulSets, DaemonSets) via their pod template spec (`spec.template.spec.tolerations`).

## 3. Click-to-Copy and Hover Raw Values

### CopyableValue component
A new reusable component (`@klados/ui` or local) that wraps detail field values.

**Behavior:**
- Clicking copies the raw value to clipboard
- Visual feedback: brief background highlight that fades out over ~300ms (no toast)
- Cursor changes to pointer on hover
- Subtle dotted underline on hover to hint at interactivity
- A small copy icon (clipboard) fades in to the right of the value on hover
- Icon is absolute-positioned or pre-reserved space to avoid layout shift
- After successful copy, icon briefly swaps to a checkmark, then fades back

**Scope:** Only values are copyable, not labels. For age fields, copies the raw ISO timestamp, not the display string.

### Hover raw values
- `age` render type: native `title` tooltip shows the full ISO timestamp
- Truncated text fields: native `title` attribute shows the full value (CSS `truncate` already clips with ellipsis)

## 4. Conditions Card — 3 Columns

Change the conditions grid from `grid-cols-1 sm:grid-cols-2` to `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3`. Each condition pill stays the same (green/gray dot, type name, reason, age). Reduces vertical space.

## 5. Container Card Accordion

Replace the current flat container card layout with structured accordion sections.

### Always-visible header
Container name, status badge, restart count, and image. Same as today.

### Accordion sections
Four collapsible sections below the header:

| Section | Default State | Content |
|---|---|---|
| Resources | Expanded | CPU/Mem/Disk request-limit pairs (existing layout) |
| Ports | Expanded | Port buttons with port-forward click (existing layout) |
| Environment | Collapsed | Count in header, expands to key/value grid |
| Mounts | Collapsed | Count in header, expands to mount list |

### Rules
- Sections with no data are hidden entirely (no empty collapsible header)
- The full-width section header row is the click target (not just the arrow)
- Collapsed headers for Environment and Mounts show the item count (e.g., "Environment (4)")

## Files to Modify

| File | Change |
|---|---|
| `internal/resource/builtin.go` | Add new overviewFields for pods, deployments, statefulsets, daemonsets |
| `frontend/src/lib/components/panels/OverviewPanel.svelte` | Tolerations section, accordion containers, conditions 3-col, CopyableValue integration |
| `packages/ui/src/lib/` (new) | `CopyableValue.svelte` component |
