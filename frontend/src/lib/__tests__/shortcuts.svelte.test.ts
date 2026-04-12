import {describe, it, expect, vi, beforeEach} from "vitest";
import {ShortcutStore} from "$lib/stores/shortcuts.svelte";

function makeKeyEvent(overrides: Partial<KeyboardEvent> = {}): KeyboardEvent {
  return {
    ctrlKey: false,
    altKey: false,
    shiftKey: false,
    metaKey: false,
    key: "",
    preventDefault: vi.fn(),
    ...overrides,
  } as unknown as KeyboardEvent;
}

describe("ShortcutStore", () => {
  let store: ShortcutStore;

  beforeEach(() => {
    store = new ShortcutStore();
  });

  it("starts in normal mode", () => {
    expect(store.focusMode).toBe("normal");
  });

  it("setMode changes focusMode", () => {
    store.setMode("terminal");
    expect(store.focusMode).toBe("terminal");
    store.setMode("editor");
    expect(store.focusMode).toBe("editor");
    store.setMode("normal");
    expect(store.focusMode).toBe("normal");
  });

  it("fires registered shortcut in normal mode", () => {
    const action = vi.fn();
    store.register({id: "test", keys: "Control+k", action, description: "Test"});
    const e = makeKeyEvent({ctrlKey: true, key: "k"});
    store.dispatch(e);
    expect(action).toHaveBeenCalledOnce();
    expect(e.preventDefault).toHaveBeenCalledOnce();
  });

  it("does not fire shortcut in terminal mode", () => {
    const action = vi.fn();
    store.register({id: "test", keys: "Control+k", action, description: "Test"});
    store.setMode("terminal");
    const e = makeKeyEvent({ctrlKey: true, key: "k"});
    store.dispatch(e);
    expect(action).not.toHaveBeenCalled();
  });

  it("escape chord exits terminal mode", () => {
    store.setMode("terminal");
    const e = makeKeyEvent({ctrlKey: true, shiftKey: true, key: "Escape"});
    store.dispatch(e);
    expect(store.focusMode).toBe("normal");
    expect(e.preventDefault).toHaveBeenCalledOnce();
  });

  it("does not fire normal-only shortcut in editor mode", () => {
    const action = vi.fn();
    store.register({id: "test", keys: "Control+z", action, description: "Test"});
    store.setMode("editor");
    store.dispatch(makeKeyEvent({ctrlKey: true, key: "z"}));
    expect(action).not.toHaveBeenCalled();
  });

  it("fires shortcut with editor mode when explicitly listed", () => {
    const action = vi.fn();
    store.register({id: "test", keys: "Control+k", action, description: "Test", modes: ["normal", "editor"]});
    store.setMode("editor");
    const e = makeKeyEvent({ctrlKey: true, key: "k"});
    store.dispatch(e);
    expect(action).toHaveBeenCalledOnce();
  });

  it("unregister removes shortcut", () => {
    const action = vi.fn();
    store.register({id: "test", keys: "Control+k", action, description: "Test"});
    store.unregister("test");
    store.dispatch(makeKeyEvent({ctrlKey: true, key: "k"}));
    expect(action).not.toHaveBeenCalled();
  });

  it("re-registering same id replaces shortcut", () => {
    const action1 = vi.fn();
    const action2 = vi.fn();
    store.register({id: "test", keys: "Control+k", action: action1, description: "Test"});
    store.register({id: "test", keys: "Control+k", action: action2, description: "Test v2"});
    store.dispatch(makeKeyEvent({ctrlKey: true, key: "k"}));
    expect(action1).not.toHaveBeenCalled();
    expect(action2).toHaveBeenCalledOnce();
  });
});
