import {vi} from "vitest";

// xterm uses HTMLCanvasElement which jsdom doesn't support
vi.mock("@xterm/xterm", () => ({
  Terminal: vi.fn().mockImplementation(() => ({
    open: vi.fn(),
    write: vi.fn(),
    clear: vi.fn(),
    dispose: vi.fn(),
    loadAddon: vi.fn(),
    onData: vi.fn(),
    onResize: vi.fn(),
    onTitleChange: vi.fn(),
    textarea: null,
    cols: 80,
    rows: 24,
  })),
}));
vi.mock("@xterm/addon-webgl", () => ({WebglAddon: vi.fn()}));
vi.mock("@xterm/addon-fit", () => ({FitAddon: vi.fn().mockImplementation(() => ({fit: vi.fn(), activate: vi.fn()}))}));
vi.mock("@xterm/addon-clipboard", () => ({ClipboardAddon: vi.fn()}));
vi.mock("@xterm/addon-search", () => ({SearchAddon: vi.fn()}));
vi.mock("@xterm/addon-web-links", () => ({WebLinksAddon: vi.fn()}));

// codemirror-json-schema/yaml has a missing internal file in the installed dist
vi.mock("codemirror-json-schema/yaml", () => ({yamlSchema: vi.fn(() => [])}));

// jsdom doesn't provide matchMedia
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// jsdom doesn't provide ResizeObserver
if (!globalThis.ResizeObserver) {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
}

vi.mock("@wailsio/runtime", () => ({
  Events: {
    On: vi.fn().mockReturnValue(vi.fn()),
    Off: vi.fn(),
    Emit: vi.fn(),
  },
  Call: {
    ByID: vi.fn().mockResolvedValue(undefined),
  },
  CancellablePromise: Promise,
  Create: {
    Array: (fn: any) => (arr: any[]) => arr?.map(fn) ?? [],
    Any: (v: any) => v,
    Nullable: (fn: any) => (v: any) => (v == null ? null : fn(v)),
    Map: (_kfn: any, vfn: any) => (obj: any) => {
      if (obj == null) {
        return {};
      }
      const result: any = {};
      for (const k of Object.keys(obj)) {
        result[k] = vfn(obj[k]);
      }
      return result;
    },
  },
}));
