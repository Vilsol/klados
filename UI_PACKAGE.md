# @klados/ui Component Library + Docs Site

## Context

Klados needs a publishable `@klados/ui` Svelte 5 component package extracted from `frontend/src/lib/components/`, consumed by the Wails frontend and plugin UIs. Alongside it, a Storybook 10.x docs + playground site at `apps/docs/`. The workspace is being migrated from npm to pnpm workspaces. The existing `@klados/plugin-ui` package is renamed to `@klados/plugin-sdk`.

## Decisions

**`@sveltejs/package` for building `@klados/ui`**
The `svelte-package` CLI is the standard for Svelte component libraries. It handles `.svelte` files, preserves them in output for consumers to compile, and generates a proper `exports` map. Requires only a `svelte.config.js` — no full SvelteKit app.

**Ship a `theme.css` file from `@klados/ui`**
Custom Tailwind v4 tokens (`bg`, `fg`, `muted`, `surface`, `surface-hover`, `border`, `accent`, `destructive`) are defined in a single CSS file exported from the package. Consumers (frontend, plugin UIs, Storybook) import it once. This keeps the token source of truth inside `@klados/ui`.

**Storybook lives in `apps/docs/`**
It's a deployable app, not a library. Separate from `packages/`.

**`@klados/plugin-sdk` replaces `@klados/plugin-ui`**
With actual UI components now in `@klados/ui`, the old name was misleading. The package contains manifest/context types and `defineKladosPlugin()` — a build-time SDK, not a UI package.

**Generic components move to `packages/ui/src/lib/`; app-specific stay in `frontend/`**
Frontend imports generic components from `@klados/ui` via workspace protocol.

## Rejected Alternatives

**Vite lib mode for `@klados/ui`**
Would require manually configuring exports, `vite-plugin-dts`, and `.svelte` file handling. `@sveltejs/package` solves all of that out of the box.

**Histoire for docs site**
Svelte 5 + Vite 8 support status unconfirmed. Storybook 10.3 explicitly supports both.

## Library Selections

| Library | Purpose | Why chosen | Alternatives considered |
|---------|---------|------------|------------------------|
| `@sveltejs/package` | Build `@klados/ui` | Standard Svelte library packaging tool | Vite lib mode |
| Storybook 10.x (`@storybook/svelte-vite`) | Docs + playground | Explicit Vite 8, Tailwind v4, Svelte support in 10.3 | Histoire (Svelte 5 + Vite 8 status unconfirmed) |
| pnpm workspaces | Monorepo management | User preference; better for workspace linking than npm | npm workspaces |

## Potential Gotchas

- **`@sveltejs/package` needs `svelte.config.js`** in the package root even without SvelteKit. Without it, the CLI won't know how to process `.svelte` files.
- **Tailwind v4 in Storybook**: `@storybook/addon-pseudo-states` 10.3 supports Tailwind v4, but Storybook will need `@tailwindcss/vite` in its vite config (same as the frontend).
- **Shared Svelte runtime**: When Storybook imports `@klados/ui` components, there's only one Svelte instance (Storybook's own build), so the shared-runtime complexity from the plugin system doesn't apply here.
- **`bits-ui` peer dep**: Components using `bits-ui` primitives require it as a peer dep of `@klados/ui`. Plugin authors who use those components will need it in their project too.
- **`packages/sdk/`** currently exists with an empty `package.json` — needs to be removed or repurposed to avoid confusion.
- **`plugin-ui` rename**: Any existing references to `@klados/plugin-ui` in `examples/plugin-node-annotator/` need updating to `@klados/plugin-sdk`.

## Implementation Details

### Workspace structure

```
pnpm-workspace.yaml
package.json           ← workspaces root (no deps, just scripts)
frontend/              ← Wails app, depends on @klados/ui
packages/
  ui/                  ← @klados/ui
  plugin-sdk/          ← @klados/plugin-sdk (renamed from plugin-ui)
apps/
  docs/                ← Storybook
examples/
  plugin-node-annotator/
```

### `pnpm-workspace.yaml`

```yaml
packages:
  - 'frontend'
  - 'packages/*'
  - 'apps/*'
```

### `packages/ui/package.json`

```json
{
  "name": "@klados/ui",
  "version": "0.1.0",
  "type": "module",
  "svelte": "./package/index.js",
  "exports": {
    ".": {
      "svelte": "./package/index.js",
      "default": "./package/index.js"
    },
    "./theme.css": "./package/theme.css"
  },
  "files": ["package"],
  "scripts": {
    "dev": "svelte-package --watch",
    "build": "svelte-package"
  },
  "peerDependencies": {
    "svelte": "^5",
    "bits-ui": "^1",
    "tailwindcss": "^4"
  },
  "devDependencies": {
    "@sveltejs/package": "^2",
    "svelte": "^5"
  }
}
```

### `packages/ui/svelte.config.js`

```js
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  preprocess: vitePreprocess(),
}
```

### `packages/ui/src/lib/index.ts` — barrel export

```ts
export { default as Button }          from './Button.svelte'
export { default as Select }          from './Select.svelte'
export { default as Badge }           from './Badge.svelte'
export { default as Input }           from './Input.svelte'
export { default as Icon }            from './Icon.svelte'
export { default as Tooltip }         from './Tooltip.svelte'
export { default as Dialog }          from './Dialog.svelte'
export { default as DropdownMenu }    from './DropdownMenu.svelte'
export { default as CodeBlock }       from './CodeBlock.svelte'
export { default as Notification }    from './Notification.svelte'
export { default as TabBar }          from './TabBar.svelte'
export { default as ConfirmDialog }   from './ConfirmDialog.svelte'
export { default as DetailDrawer }    from './DetailDrawer.svelte'
export { default as LogViewer }       from './LogViewer.svelte'
export { default as VirtualLogViewer } from './VirtualLogViewer.svelte'
export { default as Terminal }        from './Terminal.svelte'
export { default as YAMLEditor }      from './YAMLEditor.svelte'
```

### `packages/ui/src/lib/theme.css` — token definitions

```css
@layer base {
  :root {
    --color-bg: ...;
    --color-fg: ...;
    /* all Tailwind v4 custom tokens: bg, fg, muted, border, accent, surface, surface-hover, destructive */
  }
  .dark { ... }
}
```

### Frontend dependency (`frontend/package.json`)

```json
{
  "dependencies": {
    "@klados/ui": "workspace:*"
  }
}
```

### Component split

**Extract to `packages/ui/src/lib/`** (move from `frontend/src/lib/components/`):
- `Select`
- `CodeBlock`
- `Notification`
- `TabBar`
- `ConfirmDialog`
- `DetailDrawer`
- `LogViewer`
- `VirtualLogViewer`
- `Terminal`
- `YAMLEditor`

**New components to add to `@klados/ui`**:
- `Button`
- `Icon` (lucide-svelte wrapper)
- `Badge`
- `Input`
- `Tooltip`
- `Dialog` (headless via bits-ui)
- `DropdownMenu`

**App-specific — stay in `frontend/src/lib/components/`**:
- `Sidebar`
- `Header`
- `ResourceList`
- `ResourceDetail`
- `CommandPalette`
- `PortForwardDialog`
- `CreateResourceDialog`
- `KubeconfigImportDialog`
- `ConnectionIndicator`
- `Layout`

### `apps/docs/` Storybook structure

```
apps/docs/
  .storybook/
    main.ts        ← framework: @storybook/svelte-vite, addons
    preview.ts     ← import @klados/ui/theme.css, configure viewport
  src/stories/
    Button.stories.ts
    Select.stories.ts
    Badge.stories.ts
    ...
  package.json
  vite.config.ts
```

### `apps/docs/package.json` (key parts)

```json
{
  "name": "docs",
  "private": true,
  "scripts": {
    "storybook": "storybook dev -p 6006",
    "build": "storybook build"
  },
  "devDependencies": {
    "storybook": "^10",
    "@storybook/svelte-vite": "^10",
    "@storybook/addon-essentials": "^10",
    "@tailwindcss/vite": "^4",
    "tailwindcss": "^4",
    "svelte": "^5",
    "@klados/ui": "workspace:*"
  }
}
```

### Story format (CSF)

```ts
// Button.stories.ts
import type { Meta, StoryObj } from '@storybook/svelte'
import { Button } from '@klados/ui'

const meta = {
  component: Button,
  tags: ['autodocs'],
} satisfies Meta<typeof Button>

export default meta
type Story = StoryObj<typeof meta>

export const Primary: Story = {
  args: { variant: 'primary', label: 'Click me' }
}

export const Destructive: Story = {
  args: { variant: 'destructive', label: 'Delete' }
}
```

### `apps/docs/.storybook/preview.ts`

```ts
import '@klados/ui/theme.css'
import type { Preview } from '@storybook/svelte'

const preview: Preview = {
  parameters: {
    backgrounds: { disable: true },  // use theme.css tokens instead
  },
}

export default preview
```

## Definition of Done

- `pnpm install` at repo root links all workspaces cleanly
- `pnpm --filter @klados/ui build` produces `package/` with correct exports
- `frontend` imports generic components from `@klados/ui` (not from `./components/`)
- `apps/docs` starts with `pnpm storybook` and renders all extracted + new components
- `@klados/plugin-sdk` is the renamed package; `examples/plugin-node-annotator` references updated
- `packages/sdk/` removed or repurposed
- All existing frontend tests pass after the component move
