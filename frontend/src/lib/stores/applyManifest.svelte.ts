let open = $state(false)

export const applyManifestStore = {
  get open() {
    return open
  },
  set open(v: boolean) {
    open = v
  },
  openDialog() {
    open = true
  },
  close() {
    open = false
  },
}
