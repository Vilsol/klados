package exec

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/slox"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Vilsol/klados/internal/cluster"
)

type fakeProvider struct {
	conn *cluster.Connection
	err  error
}

func (f *fakeProvider) GetConnection(_ string) (*cluster.Connection, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.conn, nil
}

func newTestManager(p ConnectionProvider) *Manager {
	ctx := slox.Into(context.Background(), slog.Default())
	return NewManager(p, ctx)
}

func fakeConn() *cluster.Connection {
	return &cluster.Connection{Clientset: fake.NewSimpleClientset()}
}

func TestOpenSession_UniqueIDs(t *testing.T) {
	m := newTestManager(&fakeProvider{conn: fakeConn()})
	id1, err := m.OpenSession("ctx1", "ns", "pod", "container", "bash")
	testza.AssertNoError(t, err)

	id2, err := m.OpenSession("ctx1", "ns", "pod", "container", "bash")
	testza.AssertNoError(t, err)

	testza.AssertNotEqual(t, id1, id2)
	testza.AssertLen(t, id1, 32)
	testza.AssertLen(t, id2, 32)

	m.StopAll()
}

func TestOpenSession_UnknownContext(t *testing.T) {
	m := newTestManager(&fakeProvider{err: fmt.Errorf("not connected")})
	_, err := m.OpenSession("ctx1", "ns", "pod", "container", "bash")
	testza.AssertNotNil(t, err)
}

func TestCloseSession_Cleanup(t *testing.T) {
	m := newTestManager(&fakeProvider{conn: fakeConn()})
	id, err := m.OpenSession("ctx1", "ns", "pod", "container", "bash")
	testza.AssertNoError(t, err)

	m.mu.Lock()
	_, exists := m.sessions[id]
	m.mu.Unlock()
	testza.AssertTrue(t, exists)

	m.CloseSession(id)

	m.mu.Lock()
	_, exists = m.sessions[id]
	m.mu.Unlock()
	testza.AssertFalse(t, exists)
}

func TestStopAll_ClearsAll(t *testing.T) {
	m := newTestManager(&fakeProvider{conn: fakeConn()})
	_, err := m.OpenSession("ctx1", "ns", "pod1", "c1", "bash")
	testza.AssertNoError(t, err)
	_, err = m.OpenSession("ctx1", "ns", "pod2", "c2", "sh")
	testza.AssertNoError(t, err)

	m.mu.Lock()
	count := len(m.sessions)
	m.mu.Unlock()
	testza.AssertEqual(t, 2, count)

	m.StopAll()

	m.mu.Lock()
	count = len(m.sessions)
	m.mu.Unlock()
	testza.AssertEqual(t, 0, count)
}

func TestCloseSession_NonExistent(t *testing.T) {
	m := newTestManager(&fakeProvider{conn: fakeConn()})
	m.CloseSession("nonexistent-id")
}
