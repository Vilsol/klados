# node-annotator

Example klados plugin that enriches Kubernetes nodes with taint and readiness information.

## What it adds

### List columns (Nodes view)

Replaces the default node descriptor with three columns:

| Column | Value |
|--------|-------|
| **Status** | `Ready`, `NotReady`, `NotReady:<reason>`, or `Unknown` — derived from the node's `Ready` condition |
| **Taints** | Count of taints on `spec.taints` |
| **Age** | Node creation timestamp |

### Detail tab (node drawer → "Annotations")

Opens an **Annotations** tab on any node's detail drawer showing:

- **This Node** — readiness status and taint count for the selected node, with individual taint badges (`key=value:effect`)
- **All Nodes** — a table of every node in the cluster with their readiness and taint count

### Metric charts (node drawer → "Metrics")

When Prometheus is available, adds three charts below the built-in CPU/Memory graphs:

| Chart | Query | Unit |
|-------|-------|------|
| **Disk Available** | Root filesystem free ratio | ratio (0–1) |
| **Network Receive** | Inbound bytes/s across all non-loopback interfaces | bytes |
| **Pod Count** | Running pods on the node (from kubelet) | count |

These are declarative — no Wasm or UI code required, just the `metrics` field in `manifest.json`.

## Building

Requires [mise](https://mise.jdx.dev/) and [TinyGo](https://tinygo.org/).

```bash
# Build TinyGo wasm + UI
mise run build:tinygo
mise run build:ui

# Build everything (including standard Go wasm)
mise run build

# Run enricher unit tests
mise run test

# Build and pack into an OCI archive for sideloading
mise run pack
```

> **Note:** The plugin uses the TinyGo build (`dist/plugin-tiny.wasm`). The standard Go build (`dist/plugin.wasm`) is provided for reference but is not used at runtime — standard Go WASM requires reactor mode (`_initialize`) which Go does not yet generate.

## Installing

**Dev (directory copy):**
```bash
cp -r . ~/.local/share/klados/plugins/node-annotator
```

**Sideload (OCI archive):** run `mise run pack` then install `node-annotator-0.1.0.oci.tar.gz` via the klados Plugin Management page.

## Permissions

The plugin declares the following permissions in `manifest.json`:

- `resources`: `core/v1/nodes` — list, get, watch (used by the detail tab to load all nodes)
- `storage`: key/value storage (declared, not currently used)
- `events`: cluster event bus (declared, not currently used)
