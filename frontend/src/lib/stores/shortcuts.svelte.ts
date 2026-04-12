import {preferencesStore} from "./preferences.svelte";

export type FocusMode = "normal" | "terminal" | "editor";

export interface ShortcutDef {
  id: string;
  // Key combo string: modifiers first, then key. E.g. 'Control+k', 'Control+Shift+Escape'
  keys: string;
  action: () => void;
  description: string;
  // Modes in which this shortcut fires. Defaults to ['normal'].
  modes?: FocusMode[];
}

function buildKeyCombo(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey) parts.push("Control");
  if (e.altKey) parts.push("Alt");
  if (e.shiftKey) parts.push("Shift");
  if (e.metaKey) parts.push("Meta");
  if (!["Control", "Alt", "Shift", "Meta"].includes(e.key)) {
    parts.push(e.key);
  }
  return parts.join("+");
}

export class ShortcutStore {
  focusMode = $state<FocusMode>("normal");
  private _shortcuts: ShortcutDef[] = [];

  setMode(mode: FocusMode) {
    this.focusMode = mode;
  }

  register(def: ShortcutDef) {
    const idx = this._shortcuts.findIndex((s) => s.id === def.id);
    if (idx >= 0) this._shortcuts[idx] = def;
    else this._shortcuts.push(def);
  }

  unregister(id: string) {
    const idx = this._shortcuts.findIndex((s) => s.id === id);
    if (idx >= 0) this._shortcuts.splice(idx, 1);
  }

  getEffectiveKeys(def: ShortcutDef): string {
    const override = preferencesStore.getKeybinding(def.id);
    return override ?? def.keys;
  }

  getAll(): ShortcutDef[] {
    return [...this._shortcuts];
  }

  dispatch(e: KeyboardEvent) {
    const combo = buildKeyCombo(e);

    if (this.focusMode === "terminal") {
      // Only the escape chord exits terminal capture mode
      if (combo === "Control+Shift+Escape") {
        this.focusMode = "normal";
        e.preventDefault();
      }
      return;
    }

    for (const def of this._shortcuts) {
      if (combo !== this.getEffectiveKeys(def)) continue;
      const modes = def.modes ?? ["normal"];
      if (!modes.includes(this.focusMode)) continue;
      e.preventDefault();
      def.action();
      return;
    }
  }
}

export const shortcutStore = new ShortcutStore();
