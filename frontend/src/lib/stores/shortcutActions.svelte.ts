// Reactive action bus for keyboard shortcuts that need to trigger component-local behavior.
// Shortcut handlers increment a counter; components react via $effect.
class ShortcutActions {
  focusSearch = $state(0);
  selectAll = $state(0);
  deleteSelected = $state(0);
  refreshList = $state(0);
  confirmDialog = $state(0);
  copyResourceNames = $state(0);
}

export const shortcutActions = new ShortcutActions();
