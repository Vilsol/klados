package services

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/session"
)

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 10}))
}

func writeTestKubeconfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config")
	content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
    namespace: default
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: fake-token
`
	testza.AssertNoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func setupClusterService(t *testing.T) (*ClusterService, *session.Session) {
	t.Helper()
	p := writeTestKubeconfig(t)
	t.Setenv("KUBECONFIG", p)

	mgr := cluster.NewManager(func(string, any) {}, nil, context.Background())
	testza.AssertNoError(t, mgr.LoadKubeconfigs(nil))

	sess := &session.Session{
		ConnectedClusters: []string{},
		ActiveNamespaces:  map[string]string{},
	}

	appSvc := &AppService{clusterMgr: mgr}
	svc := &ClusterService{
		appService: appSvc,
		session:    sess,
	}

	return svc, sess
}

func TestClusterService_ListContexts(t *testing.T) {
	svc, _ := setupClusterService(t)

	contexts := svc.ListContexts()
	testza.AssertTrue(t, len(contexts) > 0)
	testza.AssertEqual(t, "test-context", contexts[0].Name)
}

func TestClusterService_SwitchNamespace(t *testing.T) {
	svc, sess := setupClusterService(t)

	testza.AssertNoError(t, svc.SwitchNamespace("test-context", "kube-system"))

	testza.AssertEqual(t, "kube-system", sess.ActiveNamespaces["test-context"])
}

func TestClusterService_GetStatusDisconnected(t *testing.T) {
	svc, _ := setupClusterService(t)

	status := svc.GetStatus("test-context")
	testza.AssertEqual(t, cluster.StatusDisconnected, status)
}

func TestClusterService_DisconnectUpdatesSession(t *testing.T) {
	svc, sess := setupClusterService(t)

	sess.ConnectedClusters = []string{"test-context", "other"}

	testza.AssertNoError(t, svc.Disconnect("test-context"))

	testza.AssertEqual(t, []string{"other"}, sess.ConnectedClusters)
}

func TestAppendUnique(t *testing.T) {
	result := appendUnique([]string{"a", "b"}, "b")
	testza.AssertEqual(t, []string{"a", "b"}, result)

	result = appendUnique([]string{"a", "b"}, "c")
	testza.AssertEqual(t, []string{"a", "b", "c"}, result)
}

func TestRemoveString(t *testing.T) {
	result := removeString([]string{"a", "b", "c"}, "b")
	testza.AssertEqual(t, []string{"a", "c"}, result)

	result = removeString([]string{"a"}, "x")
	testza.AssertEqual(t, []string{"a"}, result)
}
