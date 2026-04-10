// Global state for the Create Resource dialog.
// Allows the command palette to open it without a resource list page context.

let open = $state(false)
let gvr = $state<string | undefined>(undefined)
let onsuccess = $state<(() => void) | undefined>(undefined)

export const createResourceStore = {
  get open() {
    return open
  },
  set open(v: boolean) {
    open = v
  },
  get gvr() {
    return gvr
  },
  get onsuccess() {
    return onsuccess
  },
  openDialog(opts?: { gvr?: string; onsuccess?: () => void }) {
    gvr = opts?.gvr
    onsuccess = opts?.onsuccess
    open = true
  },
  close() {
    open = false
  },
}
