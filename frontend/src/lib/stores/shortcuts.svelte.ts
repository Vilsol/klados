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
  // Grouping category for the shortcut help dialog.
  category?: string;
  // If true, this shortcut is hidden from the help dialog (e.g. alias bindings).
  hidden?: boolean;
}

function buildKeyCombo(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey) {
    parts.push("Control");
  }
  if (e.altKey) {
    parts.push("Alt");
  }
  if (e.shiftKey) {
    // Don't include Shift for symbol characters — Shift is already encoded
    // in the key value (e.g., Shift+/ produces e.key="?", so the combo
    // should be "?" not "Shift+?"). Only include Shift for letters/digits
    // and non-printable keys.
    const isShiftedSymbol = e.key.length === 1 && !/[a-zA-Z0-9]/.test(e.key);
    if (!isShiftedSymbol) {
      parts.push("Shift");
    }
  }
  if (e.metaKey) {
    parts.push("Meta");
  }
  if (!["Control", "Alt", "Shift", "Meta"].includes(e.key)) {
    parts.push(e.key);
  }
  return parts.join("+");
}

export class ShortcutStore {
  focusMode = $state<FocusMode>("normal");
  private readonly _shortcuts: ShortcutDef[] = [];

  setMode(mode: FocusMode) {
    this.focusMode = mode;
  }

  register(def: ShortcutDef) {
    const idx = this._shortcuts.findIndex((s) => s.id === def.id);
    if (idx >= 0) {
      this._shortcuts[idx] = def;
    } else {
      this._shortcuts.push(def);
    }
  }

  unregister(id: string) {
    const idx = this._shortcuts.findIndex((s) => s.id === id);
    if (idx >= 0) {
      this._shortcuts.splice(idx, 1);
    }
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

    // Skip shortcuts when focus is on a text input and the key combo is a
    // standard text-editing operation (select-all, copy, paste, cut, undo, redo)
    // or an unmodified single key.
    const el = document.activeElement;
    const inTextInput =
      el instanceof HTMLInputElement ||
      el instanceof HTMLTextAreaElement ||
      (el instanceof HTMLElement && el.isContentEditable);

    if (inTextInput) {
      const hasModifier = e.ctrlKey || e.altKey || e.metaKey;
      if (!hasModifier) {
        return;
      }
      const textEditKeys = new Set(["a", "c", "v", "x", "z"]);
      if ((e.ctrlKey || e.metaKey) && !e.altKey && textEditKeys.has(e.key.toLowerCase())) {
        return;
      }
    }

    for (const def of this._shortcuts) {
      if (combo !== this.getEffectiveKeys(def)) {
        continue;
      }
      const modes = def.modes ?? ["normal"];
      if (!modes.includes(this.focusMode)) {
        continue;
      }
      e.preventDefault();
      def.action();
      return;
    }
  }
}

export const shortcutStore = new ShortcutStore();
