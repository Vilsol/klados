//go:build !release

package cluster

// SetConnectionForTest directly injects a Connection into the Manager for unit testing.
func (m *Manager) SetConnectionForTest(contextName string, conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[contextName] = conn
}

// SetDiscoveredResourcesForTest pre-populates the discoveredResources map for unit testing.
func (m *Manager) SetDiscoveredResourcesForTest(contextName string, resources []APIResource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.discoveredResources == nil {
		m.discoveredResources = map[string][]APIResource{}
	}
	m.discoveredResources[contextName] = resources
}
