# Phase 5 — Frontend Registry UI

Add the registry install input and inline credential prompt to the plugin management page,
completing the end-to-end GUI flow for installing plugins from an OCI registry.

## First Action

Open `frontend/src/routes/PluginManagement.svelte` and find the section where local plugin
install is handled (look for `InstallPlugin` call and the surrounding UI). Understand the
existing install flow — how loading state is managed, how errors surface as toasts, and how the
plugin list is refreshed — before adding anything. Your registry install section follows the
same patterns.

## Context

Phase 4 extended `InstallPlugin` to handle `oci://` references and added `SaveRegistryCredentials`
and `AddInsecureRegistry` as new Wails-bound methods with regenerated bindings. The backend is
complete. This phase adds the UI: a registry ref input field, a loading state, and — when the
backend returns `ErrAuthRequired` — an inline credential form that saves credentials to
`~/.docker/config.json` and retries the install automatically.

## Files to Read

- `frontend/src/routes/PluginManagement.svelte` — find the existing install UI block; understand
  `$state` variables for loading/error, how `InstallPlugin` is called, and how toast
  notifications work — your registry section uses the same patterns
- `frontend/bindings/github.com/Vilsol/klados/internal/services/pluginservice.js` (or `.ts`) —
  find `SaveRegistryCredentials` and `AddInsecureRegistry` to get the exact import paths and
  function signatures
- `frontend/src/lib/__tests__/wails-mock.ts` — verify `SaveRegistryCredentials` and
  `AddInsecureRegistry` stubs are present (added in Phase 4); add them if missing before
  writing tests
- `frontend/src/lib/__tests__/PluginManagement.svelte.test.ts` (if it exists) — understand
  existing test structure; if it doesn't exist, look at `Sidebar.svelte.test.ts` for the
  mocking pattern to replicate

## What Exists

**From Phase 4:**
- `PluginService.InstallPlugin("oci://...")` — works, returns `ErrAuthRequired` on auth failure
  with message `"authentication required"`
- `PluginService.SaveRegistryCredentials(host, username, password)` — saves to `~/.docker/config.json`
- `PluginService.AddInsecureRegistry(host)` — persists to config
- Regenerated Wails bindings with all three methods

**Baseline:**
- `frontend/src/routes/PluginManagement.svelte` — existing plugin management UI with local install
- `frontend/src/lib/stores/notification.svelte.ts` — toast queue used for success/error messages
- `frontend/src/lib/__tests__/wails-mock.ts` — mock registry for Wails service calls

## Deliverables

1. `frontend/src/routes/PluginManagement.svelte` updated with a "Install from registry" section:

   **Input state**
   - `registryRef: string` — the raw text field value
   - `registryLoading: boolean` — true while `InstallPlugin` is in flight
   - `registryError: string` — non-auth error message (shown inline)
   - `showAuthForm: boolean` — true after `ErrAuthRequired` response
   - `authHost: string` — registry host extracted from the ref (pre-fills the form label)
   - `authUsername: string`, `authPassword: string` — credential form fields
   - `authInsecure: boolean` — insecure checkbox

   **Input field behaviour**
   - Accepts bare refs (`ghcr.io/foo/bar:v1`) or `oci://`-prefixed refs
   - On submit: normalise to `oci://` prefix if not already present, call
     `InstallPlugin(normalisedRef)`
   - On success: clear input, show success toast, refresh plugin list
   - On error where `err.message.includes("authentication required")`: set `showAuthForm = true`,
     set `authHost` by parsing the first `/`-delimited segment after stripping `oci://`
   - On any other error: set `registryError` and show it inline

   **Credential form** (rendered when `showAuthForm` is true, below the input field)
   - Shows `authHost` as a label so the user knows which registry
   - Username field bound to `authUsername`
   - Password/token field bound to `authPassword` (single field; label says "Password or token")
   - "Insecure (HTTP)" checkbox bound to `authInsecure`
   - Submit button: calls `SaveRegistryCredentials(authHost, authUsername, authPassword)`, then
     if `authInsecure` calls `AddInsecureRegistry(authHost)`, then retries `InstallPlugin`
   - On retry success: hide auth form, clear fields, show success toast
   - On retry auth failure: show persistent error "Credentials rejected — verify and try again"

2. Bindings imports in `PluginManagement.svelte` updated to include `SaveRegistryCredentials`
   and `AddInsecureRegistry` from the regenerated `.js` file (use `.js` extension per Wails v3
   alpha.74 convention)

3. `frontend/src/lib/__tests__/PluginManagement.svelte.test.ts` with:
   - Happy path: `InstallPlugin` mock resolves → toast shown, plugin list refreshed
   - Auth failure path: `InstallPlugin` rejects with `{ message: "authentication required" }` →
     `showAuthForm` becomes true, `authHost` is correctly parsed from the ref
   - Credential submit: `SaveRegistryCredentials` called with correct args, `InstallPlugin`
     retried with same normalised ref
   - Insecure checkbox: `AddInsecureRegistry` called before retry when checked
   - `oci://` normalisation: bare ref `ghcr.io/foo/bar:v1` is submitted as
     `oci://ghcr.io/foo/bar:v1` to `InstallPlugin`
   - Retry auth failure: error message "Credentials rejected" shown, auth form stays visible

## Tests

- **Frontend test (vitest + @testing-library/svelte)**
  All scenarios in Deliverable 3 above.
  Mock all Wails service calls in `wails-mock.ts` — do not make real network calls.

- **Manual verification**
  - Type `ghcr.io/owner/plugin:v1` and submit → normalised to `oci://...` in the call
  - Registry requiring auth → credential form appears with host pre-filled
  - Enter credentials and submit → saves to `~/.docker/config.json`, retries, installs
  - Check `insecure` and submit → `AddInsecureRegistry` called, host added to config
  - `npx vitest run` — all frontend tests pass

## Acceptance Criteria

- [ ] Typing a registry ref and submitting calls `InstallPlugin("oci://...")` (verified in test)
- [ ] Auth failure response renders the inline credential form without a full-page error
- [ ] Credential form shows the registry host parsed from the ref
- [ ] Credential submit calls `SaveRegistryCredentials` then retries `InstallPlugin` (test)
- [ ] Insecure checkbox causes `AddInsecureRegistry` to be called before retry (test)
- [ ] `npx vitest run src/lib/__tests__/PluginManagement.svelte.test.ts` passes
- [ ] `npx vitest run` passes (no regressions to other frontend tests)

## Definition of Done

A user opens the plugin management page, types `ghcr.io/foo/plugin:v1`, and clicks install.
If the registry is public, the plugin installs and appears in the list. If the registry requires
auth, an inline form appears pre-filled with the host. After entering credentials and submitting,
the credentials are saved and the install proceeds. The "insecure" checkbox enables plain-HTTP
access. All frontend tests pass.

## Known Gotchas

- **The trap**: checking `err === ErrAuthRequired` or comparing error objects from Wails
  **Why**: Wails serialises Go errors to `{ message: "...", code: ... }` JSON objects; you
  cannot do reference equality or `instanceof` checks.
  **What to do instead**: check `err.message?.includes("authentication required")` — this matches
  the exact string `ErrAuthRequired.Error()` returns, agreed in Phase 4.

- **The trap**: parsing the registry host with `oci://ghcr.io/foo/bar:v1`.split('/')[0]
  **Why**: that gives `"oci:"`, not the host.
  **What to do instead**: strip `oci://` first, then split on `/` and take index 0.
  Pattern: `ref.replace(/^oci:\/\//, '').split('/')[0]`

- **The trap**: mocking `SaveRegistryCredentials` and `AddInsecureRegistry` as named exports
  in `wails-mock.ts` without checking how the bindings file actually exports them
  **Why**: Wails v3 generates `.js` files with a specific export shape; the mock must match
  the import path and export name exactly or the component import fails at test time.
  **What to do instead**: read the regenerated bindings file first, copy the export name
  exactly, then add the stub to `wails-mock.ts`.

- **The trap**: writing `vi.mock()` factory functions that reference top-level `vi.fn()` variables
  **Why**: `vi.mock()` factories are hoisted before variable initialisation; the variable is
  `undefined` at the time the factory runs.
  **What to do instead**: use `vi.hoisted()` to define shared mock functions. This is documented
  in `cerebrum.md` under Do-Not-Repeat.

- **The trap**: mocking Svelte 5 child components as `{ default: { render: () => {} } }`
  **Why**: Svelte 5 expects components to be callable functions, not SSR render objects.
  **What to do instead**: mock as `{ default: vi.fn() }`. Documented in `cerebrum.md`.
