package exec

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"github.com/sasha-s/go-deadlock"

	"github.com/Vilsol/slox"
	fiberws "github.com/gofiber/websocket/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/Vilsol/klados/internal/cluster"
)

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)

func init() {
	_ = corev1.AddToScheme(scheme)
}

type execSession struct {
	ctxName   string
	ns        string
	podName   string
	container string
	shell     string
	cancel    context.CancelFunc
}

type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type sizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func (s *sizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s.ch
	if !ok {
		return nil
	}
	return &size
}

type ConnectionProvider interface {
	GetConnection(contextName string) (*cluster.Connection, error)
}

type Manager struct {
	mu       deadlock.Mutex
	sessions map[string]*execSession
	connMgr  ConnectionProvider
	ctx      context.Context
}

func NewManager(connMgr ConnectionProvider, ctx context.Context) *Manager {
	return &Manager{
		sessions: make(map[string]*execSession),
		connMgr:  connMgr,
		ctx:      ctx,
	}
}

func (m *Manager) OpenSession(ctxName, ns, podName, container, shell string) (string, error) {
	if _, err := m.connMgr.GetConnection(ctxName); err != nil {
		return "", fmt.Errorf("getting connection: %w", err)
	}

	id, err := newID()
	if err != nil {
		return "", err
	}

	_, cancel := context.WithCancel(context.Background())
	session := &execSession{
		ctxName:   ctxName,
		ns:        ns,
		podName:   podName,
		container: container,
		shell:     shell,
		cancel:    cancel,
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return id, nil
}

func (m *Manager) HandleConn(sessionID string, conn *fiberws.Conn) {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	m.mu.Unlock()

	if !ok {
		_ = conn.WriteMessage(fiberws.TextMessage, []byte(`{"type":"error","message":"session not found"}`))
		return
	}

	defer func() {
		m.CloseSession(sessionID)
	}()

	kconn, err := m.connMgr.GetConnection(session.ctxName)
	if err != nil {
		slox.Warn(m.ctx, "exec session lost connection", "id", sessionID, "error", err)
		return
	}

	req := kconn.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(session.podName).
		Namespace(session.ns).
		SubResource("exec").
		Param("container", session.container).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "true").
		Param("command", session.shell)

	executor, err := remotecommand.NewSPDYExecutor(kconn.Config, "POST", req.URL())
	if err != nil {
		slox.Warn(m.ctx, "exec session failed to create executor", "id", sessionID, "error", err)
		return
	}

	sq := &sizeQueue{ch: make(chan remotecommand.TerminalSize, 8)}
	defer close(sq.ch)

	stdinR, stdinW := io.Pipe()
	defer stdinW.Close()

	// Read WS messages: binary = stdin, text = resize
	go func() {
		defer stdinW.Close()
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			switch mt {
			case fiberws.BinaryMessage:
				if _, err := stdinW.Write(msg); err != nil {
					return
				}
			case fiberws.TextMessage:
				var rm resizeMsg
				if json.Unmarshal(msg, &rm) == nil && rm.Type == "resize" {
					select {
					case sq.ch <- remotecommand.TerminalSize{Width: rm.Cols, Height: rm.Rows}:
					default:
					}
				}
			}
		}
	}()

	wsWriter := newWSWriter(conn, m.ctx, sessionID)
	defer wsWriter.close()

	execCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = executor.StreamWithContext(execCtx, remotecommand.StreamOptions{
		Stdin:             stdinR,
		Stdout:            wsWriter,
		Stderr:            wsWriter,
		Tty:               true,
		TerminalSizeQueue: sq,
	})
}

func (m *Manager) CloseSession(sessionID string) {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if ok {
		session.cancel()
	}
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		session.cancel()
		delete(m.sessions, id)
	}
}

const execBufSize = 256

type wsWriter struct {
	conn      *fiberws.Conn
	ctx       context.Context
	sessionID string
	ch        chan []byte
}

func newWSWriter(conn *fiberws.Conn, ctx context.Context, sessionID string) *wsWriter {
	w := &wsWriter{conn: conn, ctx: ctx, sessionID: sessionID, ch: make(chan []byte, execBufSize)}
	go w.drain()
	return w
}

func (w *wsWriter) Write(p []byte) (int, error) {
	chunk := make([]byte, len(p))
	copy(chunk, p)
	select {
	case w.ch <- chunk:
	default:
		slox.Debug(w.ctx, "exec stdout dropped (backpressure)", "id", w.sessionID, "bytes", len(p))
	}
	return len(p), nil
}

func (w *wsWriter) drain() {
	for chunk := range w.ch {
		if err := w.conn.WriteMessage(fiberws.BinaryMessage, chunk); err != nil {
			return
		}
	}
}

func (w *wsWriter) close() {
	close(w.ch)
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
