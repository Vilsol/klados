# Shared Svelte Runtime for Plugins

## The Problem

Plugins are loaded via dynamic `import()` from a separate origin (the Fiber streaming server at `http://127.0.0.1:PORT`). If the plugin bundles its own copy of Svelte, the browser ends up with two separate module instances:

- **Host** Svelte: loaded by the Vite/Rolldown-built app bundle
- **Plugin** Svelte: bundled inside the plugin JS file

Svelte 5 stores critical runtime state in module-level variables (`current_effect`, `next_sibling_getter`, etc.). Two instances mean two separate copies of that state. Symptoms:

- `effect_orphan` — plugin's `$effect` calls `create_effect` from its own instance where `current_effect` is null (the host mounted the component tree, not the plugin's runtime)
- `next_sibling_getter.call` crash — DOM getter initialised in one instance but called from the other
- `$.from_html is not a function` — plugin's compiled code calls `$.from_html` but `$` came from the wrong instance

## The Solution: Import Map + Shared Entry

### What makes it work

1. `index.html` has an import map that points all bare Svelte specifiers to one canonical URL:
   ```html
   <script type="importmap">
   {
     "imports": {
       "svelte":                 "/plugin-shared/svelte-runtime.js",
       "svelte/internal":        "/plugin-shared/svelte-runtime.js",
       "svelte/internal/client": "/plugin-shared/svelte-runtime.js"
     }
   }
   </script>
   ```

2. The host app's bundle also imports Svelte from the same URL (enforced by the Vite plugin's `resolveId` hook — see below).

3. The browser module cache deduplicates by URL, so both the host and any dynamically loaded plugin end up using the **exact same module instance**.

### Plugin build requirement

Plugins must mark all three Svelte specifiers as external so they don't bundle their own copy:

```js
// vite.config.js (plugin project)
rollupOptions: {
  external: ['svelte', 'svelte/internal', 'svelte/internal/client'],
}
```

---

## Dev Mode (`vite dev`)

Vite pre-bundles Svelte deps to `/.vite/deps/svelte__internal__client.js?v=HASH`. The host's compiled components reference this URL directly.

`configureServer` middleware intercepts requests to `/plugin-shared/svelte-runtime.js` and serves the result of `server.transformRequest('/src/plugin-shared/svelte-runtime.ts')`. Vite rewrites the bare specifiers inside that file to the same `/.vite/deps/...` URLs the host uses. Both share one browser-cached module. ✓

**Do NOT** add a `resolveId` hook in dev mode. If the hook returns an ID for a file that doesn't exist on disk, Vite 8's pre-transform phase crashes before the server is even started:
```
Pre-transform error: Failed to load url /plugin-shared/svelte-runtime.js. Does the file exist?
```

---

## Production Build

### Rolldown export name aliasing (the hard bug)

In production, Rolldown (Vite 8's bundler) mangles export names for **internal** (non-entry) chunks. A chunk only consumed by other chunks in the same bundle gets exports renamed to single letters:

```js
export { $document as $, onMount as A, from_html as L, mount as M, ... }
```

The plugin does `import * as $ from 'svelte/internal/client'` and calls `$.from_html` — which is `undefined` because the export is named `L`.

**Entry chunks always preserve original export names.** The fix is to make `svelte-runtime.ts` an explicit Rollup/Rolldown entry, not just a manually-assigned chunk.

### Why `this.emitFile` doesn't work (Rolldown / Vite 8)

`this.emitFile({ type: 'chunk', ... })` in a plugin's `buildStart` hook is a Rollup API. Rolldown does not support it — the file simply doesn't appear in the output. Use `rollupOptions.input` instead.

### Correct production config

```ts
// vite.config.ts
build: {
  rollupOptions: {
    input: {
      index: path.resolve('./index.html'),
      'plugin-shared/svelte-runtime': path.resolve('./src/plugin-shared/svelte-runtime.ts'),
    },
    preserveEntrySignatures: 'exports-only',
    output: {
      entryFileNames(chunk) {
        if (chunk.name === 'plugin-shared/svelte-runtime') return '[name].js'
        return 'assets/[name]-[hash].js'
      },
      chunkFileNames: 'assets/[name]-[hash].js',
    },
  },
},
```

The `svelteSharedRuntime` Vite plugin's `resolveId` hook redirects all host Svelte imports to `runtimePath` (the `svelte-runtime.ts` file). The svelte-runtime.ts itself is excluded from redirection (checked by `importer === runtimePath`) so it can import the real Svelte packages. Both host and plugin end up at the same `/plugin-shared/svelte-runtime.js` URL.

### `preserveEntrySignatures: 'exports-only'`

Tells Rolldown to keep all re-exported symbols from the entry even if the host bundle doesn't use them (e.g. `from_html` is only used by plugin compiled code). Without this, unused exports are still tree-shaken away even from entry chunks.

---

## Things That Don't Work

| Approach | Why it fails |
|---|---|
| Pre-built static `public/plugin-shared/svelte-runtime.js` | Two separate module instances → same `effect_orphan` / getter crash |
| `resolveId` returning `{ id: '/plugin-shared/svelte-runtime.js', external: true }` in dev | Vite 8 pre-transform crashes if the file doesn't exist on disk |
| `configureServer` returning a 302 redirect | Wails WebKit resolves relative redirects against `wails://localhost`; browser interprets the redirect body as module text (`text/plain`) → MIME type error |
| `manualChunks` alone | Chunk is internal → exports are aliased to single letters → plugin can't find `from_html` |
| `moduleSideEffects: 'no-treeshake'` on the virtual module | Prevents tree-shaking of the module's own code but does not force all re-exported symbols from dependencies to be preserved |
| `this.emitFile({ type: 'chunk' })` in `buildStart` | Not supported by Rolldown (Vite 8); file never appears in output |

---

## Debug Infrastructure

### Terminal logging from frontend

A `POST /:token/log` route on the Fiber streaming server writes to the Go terminal logger. The frontend has a console patch in `main.ts` that forwards all `console.log/warn/error/debug` calls to this endpoint (with pre-streaming buffering). This lets you see browser-side errors — including errors inside dynamically loaded plugins — in the same terminal as Go logs.

### Diagnosing export aliasing

To confirm Rolldown is aliasing exports in a production build:

```bash
grep "from_html" dist/plugin-shared/svelte-runtime.js
# If you see: from_html as L   → aliasing is happening (internal chunk)
# If you see: from_html,       → original names preserved (entry chunk)
```
