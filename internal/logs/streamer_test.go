package logs

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

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

func newTestStreamer(p ConnectionProvider) *Streamer {
	ctx := slox.Into(context.Background(), slog.Default())
	return NewStreamer(p, ctx)
}

func fakeConn() *cluster.Connection {
	return &cluster.Connection{Clientset: fake.NewSimpleClientset()}
}

func TestStartStream_UniqueIDs(t *testing.T) {
	s := newTestStreamer(&fakeProvider{conn: fakeConn()})
	id1, err := s.StartStream("ctx1", "ns", "pod", LogOptions{})
	testza.AssertNoError(t, err)

	id2, err := s.StartStream("ctx1", "ns", "pod", LogOptions{})
	testza.AssertNoError(t, err)

	testza.AssertNotEqual(t, id1, id2)
	testza.AssertLen(t, id1, 32)
	testza.AssertLen(t, id2, 32)

	s.StopAll()
}

func TestStartStream_UnknownContext(t *testing.T) {
	s := newTestStreamer(&fakeProvider{err: fmt.Errorf("not connected")})
	_, err := s.StartStream("ctx1", "ns", "pod", LogOptions{})
	testza.AssertNotNil(t, err)
}

func TestStopStream_Cleanup(t *testing.T) {
	s := newTestStreamer(&fakeProvider{conn: fakeConn()})
	id, err := s.StartStream("ctx1", "ns", "pod", LogOptions{})
	testza.AssertNoError(t, err)

	s.mu.Lock()
	_, exists := s.streams[id]
	s.mu.Unlock()
	testza.AssertTrue(t, exists)

	s.StopStream(id)

	// readLogs goroutine exits and removes entry from map
	time.Sleep(100 * time.Millisecond)

	s.mu.Lock()
	_, exists = s.streams[id]
	s.mu.Unlock()
	testza.AssertFalse(t, exists)
}

func TestStopAll_ClearsAll(t *testing.T) {
	s := newTestStreamer(&fakeProvider{conn: fakeConn()})
	_, err := s.StartStream("ctx1", "ns", "pod1", LogOptions{})
	testza.AssertNoError(t, err)
	_, err = s.StartStream("ctx1", "ns", "pod2", LogOptions{})
	testza.AssertNoError(t, err)

	s.mu.Lock()
	count := len(s.streams)
	s.mu.Unlock()
	testza.AssertEqual(t, 2, count)

	s.StopAll()

	s.mu.Lock()
	count = len(s.streams)
	s.mu.Unlock()
	testza.AssertEqual(t, 0, count)
}

func TestStopStream_NonExistent(t *testing.T) {
	s := newTestStreamer(&fakeProvider{conn: fakeConn()})
	s.StopStream("nonexistent-id")
}
