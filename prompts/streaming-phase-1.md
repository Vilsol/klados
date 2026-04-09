# Phase 1 — Port-forward Persistence & Management Page

Add persistent, auto-reconnecting port-forwards stored in `config.json` and a dedicated management page at `/c/:ctx/port-forwards` that replaces the sidebar `+` button with a full ResourceList-based view.

## First Action

Read `internal/portforward/manager.go` to understand the existing `Manager` struct, its mutex pattern, the `Start`/`Stop`/`ListForwards` methods, and the event emission pattern (`portforward:{ctx}:{id}` and `portforward:{ctx}:updated`). Every backend change in this phase extends this file.

## Context

Port-forwards are currently ephemeral — they're lost when the app closes, and there's no centralized view of all forwards. The sidebar shows a per-context list with a `+` button that opens a dialog. This phase makes forwards persistent (saved to config, auto-reconnected on startup) and adds a management page that replaces the `+` button with a navigable route. This is the first of two streaming enhancement phases; it lands config schema and route changes that Phase 2 builds on.

## Files to Read

- `internal/portforward/manager.go` — **what to look for**: `Manager` struct fields, `Start`/`Stop`/`ListForwards` signatures, event emission via `portforward:{ctx}:{id}` and `portforward:{ctx}:updated`, the `tunnelFunc` injection pattern
- `internal/config/config.go` — **what to look for**: `Config` struct layout and JSON tags, `Load`/`Save` methods, the debounced save pattern (500ms)
- `internal/services/app_service.go` — **what to look for**: how existing port-forward RPCs (`StartPortForward`, `StopPortForward`, `ListForwards`) are wired, and the pattern for adding new Wails-bound methods
- `frontend/src/lib/components/ResourceList.svelte` — **what to look for**: how it receives data (via `ResourceStore` or props), whether it can render items without a GVR-backed watch, and the column descriptor format
- `frontend/src/lib/stores/resource.svelte.ts` — **what to look for**: `ResourceStore` constructor — does it require a GVR and watch, or can items be provided directly?
- `frontend/src/lib/components/sidebar/` — **what to look for**: the `+` button component, how it currently opens the port-forward dialog, and what navigation utility it uses
- `frontend/src/routes/routes.ts` — **what to look for**: route definition pattern, how `:ctx` param is passed to page components

## Source Documents

- `STREAMING_SPEC.md` — sections 1 (Port-forward Persistence) and 2 (Port-forward Management Page) contain the full design: config schema, Manager method signatures, virtual descriptor shape, row actions, and page layout
- `STREAMING_PHASES.md` — Phase 1 section for deliverables, tests, acceptance criteria, and handoff notes

## What Exists

- `portforward.Manager` with `Start`, `Stop`, `ListForwards` methods and per-forward / aggregate event emission
- Config system with `Load`/`Save` and 500ms debounced writes
- Sidebar port-forward list with `+` button that opens a creation dialog
- `ResourceList` component with TanStack Virtual, column descriptors, CEL expression evaluation
- `ResourceStore` for watch-backed data sources
- Wails RPC bindings for `StartPortForward`, `StopPortForward`, `ListForwards`

## Deliverables

1. `SavedPortForward` type in `internal/config/` with fields: `ID`, `Namespace`, `Resource` (e.g. `pods/my-pod`), `LocalPort`, `RemotePort`, `Enabled`
2. `PortForwards map[string][]SavedPortForward` field on `config.Config`, keyed by context name, persisted to `config.json`
3. `portforward.Manager` methods: `SaveForward`, `RemoveSavedForward`, `SetForwardEnabled`, `ListSavedForwards`, `ReconnectSaved`
4. `ReconnectSaved(ctxName)` called on cluster connect — starts all enabled saved forwards, emits per-forward error status on failure (no toast storm)
5. Wails RPC methods on `AppService` (or new `PortForwardService`): `SavePortForward`, `RemoveSavedPortForward`, `SetPortForwardEnabled`, `ListSavedPortForwards`
6. Regenerated Wails bindings
7. `/c/:ctx/port-forwards` route in `routes.ts`
8. `PortForwardPage.svelte` using `ResourceList` with a virtual descriptor (columns: Resource, Namespace, Local Port, Remote Port, Status badge, Enabled)
9. Row actions: connect/disconnect, enable/disable, remove, copy local URL
10. "New Port Forward" header button opening existing dialog
11. Sidebar `+` button changed to navigate to `/c/:ctx/port-forwards`

## Tests

- **Go unit test**
  - `SaveForward` writes to config, `ListSavedForwards` returns saved entries, `RemoveSavedForward` deletes them
  - `SetForwardEnabled(false)` marks forward disabled; `ReconnectSaved` skips disabled forwards
  - `ReconnectSaved` starts enabled forwards, emits error event (not panic) on failure (port conflict, missing pod)
  - Config round-trip: save forwards, reload config from disk, verify forwards survive restart
- **Frontend test (vitest)**
  - `PortForwardPage` renders saved forwards in ResourceList with correct columns
  - "Enable/Disable" row action calls `SetPortForwardEnabled` binding
  - "Remove" row action calls `RemoveSavedPortForward` binding
  - "New Port Forward" button opens existing dialog component
- **Manual verification**
  - Create a port-forward, close app, reopen — forward reconnects automatically
  - Disable a forward, restart — appears in list but does not connect
  - Forward on an in-use port — error status on that row, other forwards unaffected

## Acceptance Criteria

- [ ] `SavedPortForward` type exists in `internal/config/` with ID, Namespace, Resource, LocalPort, RemotePort, Enabled fields
- [ ] `config.json` persists `portForwards` map keyed by context name, survives app restart
- [ ] `portforward.Manager` exposes `SaveForward`, `RemoveSavedForward`, `SetForwardEnabled`, `ListSavedForwards`, `ReconnectSaved`
- [ ] Wails RPC methods for all CRUD operations, bindings regenerated
- [ ] On cluster connect, all enabled saved forwards for that context auto-reconnect
- [ ] Failed reconnects emit per-forward error status via `portforward:{ctx}:{id}` event (no toast storm)
- [ ] `/c/:ctx/port-forwards` page renders all saved + active forwards with status badges
- [ ] ResourceList virtual descriptor works without a GVR or watch — items provided directly
- [ ] Row actions (connect/disconnect, enable/disable, remove, copy URL) functional
- [ ] Sidebar `+` button navigates to `/c/:ctx/port-forwards` instead of opening dialog
- [ ] "New Port Forward" button on management page opens existing dialog
- [ ] Go unit tests pass for save/remove/enable/reconnect logic
- [ ] Frontend tests pass for page rendering and action bindings

## Definition of Done

A user can create a port-forward, close the app, reopen it, and see the forward automatically reconnect. The sidebar `+` button navigates to a management page listing all saved and active port-forwards with status badges. From that page, forwards can be connected/disconnected, enabled/disabled for auto-reconnect, removed, or created via the existing dialog. Port conflicts and missing resources surface as per-row error statuses, not toast storms.

## Known Gotchas

- **ResourceList assumes watch-backed data.** `ResourceList` is currently driven by `ResourceStore` which subscribes to Wails watch events for a GVR. The management page has no GVR and no watch. You'll need to either pass items directly as a prop (bypassing `ResourceStore`) or create a lightweight adapter that mimics the store interface. Document whatever pattern you use — it's the first virtual resource page.
- **Config debounce may not cover reconnect storms.** On startup with many saved forwards, rapid status changes could trigger many config writes. Verify the existing 500ms debounce in `config.Save` applies to the port-forward save path. If it doesn't, add explicit debouncing in the Manager.
- **Wails binding model collision.** The `SavedPortForward` struct will generate a model class in `frontend/bindings/`. Check that the generated name doesn't collide with existing exports in `index.js` — the project has hit this before (see cerebrum: `services/index.ts` duplicate `PluginService`).
