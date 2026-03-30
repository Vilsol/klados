//go:build integration

package cluster

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestIntegration_ConnectAndListNamespaces(t *testing.T) {
	mgr := NewManager(func(string, any) {}, nil, noopLogger())
	if err := mgr.LoadKubeconfigs(nil); err != nil {
		t.Skip("no kubeconfig available:", err)
	}

	contexts := mgr.ListContexts()
	if len(contexts) == 0 {
		t.Skip("no contexts available")
	}

	ctx := context.Background()
	ctxName := contexts[0].Name

	err := mgr.Connect(ctx, ctxName)
	testza.AssertNoError(t, err)

	nsList, err := mgr.ListNamespaces(ctx, ctxName)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, len(nsList) > 0)

	found := false
	for _, ns := range nsList {
		if ns == "default" {
			found = true
			break
		}
	}
	testza.AssertTrue(t, found, "expected 'default' namespace")

	testza.AssertNoError(t, mgr.Disconnect(ctxName))
}
