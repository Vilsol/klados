import { vi } from 'vitest'

// jsdom doesn't provide matchMedia
Object.defineProperty(window, 'matchMedia', {
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
})

vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: vi.fn(),
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
    Nullable: (fn: any) => (v: any) => v == null ? null : fn(v),
    Map: (_kfn: any, vfn: any) => (obj: any) => {
      if (obj == null) return {}
      const result: any = {}
      for (const k of Object.keys(obj)) result[k] = vfn(obj[k])
      return result
    },
  },
}))
