//go:build !release

package cluster

// SetConnectionForTest directly injects a Connection into the Manager for unit testing.
func (m *Manager) SetConnectionForTest(contextName string, conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[contextName] = conn
}
