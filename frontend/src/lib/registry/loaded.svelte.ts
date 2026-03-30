let _loaded = $state(false)

export function registryLoaded() {
  return _loaded
}

export function setRegistryLoaded() {
  _loaded = true
}
