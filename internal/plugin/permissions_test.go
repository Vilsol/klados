package plugin_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/klados/internal/plugin"
	"github.com/Vilsol/klados/internal/plugin/types"
)

func boolPtr(b bool) *bool { return &b }

func TestPermissionSet_ResourceAccess(t *testing.T) {
	perms := plugin.NewPermissionSet(&types.Permissions{
		Resources: []types.ResourcePermission{
			{Group: "apps", Version: "v1", Resource: "deployments", Verbs: []types.ResourcePermissionVerbsElem{"list", "get"}},
		},
	})

	testza.AssertTrue(t, perms.AllowsResource("apps.v1.deployments", "list"))
	testza.AssertTrue(t, perms.AllowsResource("apps.v1.deployments", "get"))
	testza.AssertFalse(t, perms.AllowsResource("apps.v1.deployments", "delete"))
	testza.AssertFalse(t, perms.AllowsResource("core.v1.pods", "list"))
}

func TestPermissionSet_Capabilities(t *testing.T) {
	t.Run("logs allowed", func(t *testing.T) {
		perms := plugin.NewPermissionSet(&types.Permissions{Logs: boolPtr(true)})
		testza.AssertTrue(t, perms.AllowsLogs())
		testza.AssertFalse(t, perms.AllowsExec())
		testza.AssertFalse(t, perms.AllowsEvents())
		testza.AssertFalse(t, perms.AllowsStorage())
	})

	t.Run("nil permissions denies all", func(t *testing.T) {
		perms := plugin.NewPermissionSet(nil)
		testza.AssertFalse(t, perms.AllowsLogs())
		testza.AssertFalse(t, perms.AllowsExec())
		testza.AssertFalse(t, perms.AllowsEvents())
		testza.AssertFalse(t, perms.AllowsStorage())
		testza.AssertFalse(t, perms.AllowsWasi("clock"))
		testza.AssertFalse(t, perms.AllowsResource("apps.v1.deployments", "list"))
	})
}

func TestPermissionSet_Wasi(t *testing.T) {
	perms := plugin.NewPermissionSet(&types.Permissions{
		Wasi: &types.WasiPermissions{Clock: boolPtr(true), Env: boolPtr(false)},
	})
	testza.AssertTrue(t, perms.AllowsWasi("clock"))
	testza.AssertFalse(t, perms.AllowsWasi("env"))
	testza.AssertFalse(t, perms.AllowsWasi("filesystem"))
	testza.AssertFalse(t, perms.AllowsWasi("network"))
}

func TestCheckPermission(t *testing.T) {
	perms := plugin.NewPermissionSet(&types.Permissions{
		Resources: []types.ResourcePermission{
			{Group: "apps", Version: "v1", Resource: "deployments", Verbs: []types.ResourcePermissionVerbsElem{"list"}},
		},
		Logs: boolPtr(true),
	})

	testza.AssertNil(t, plugin.CheckPermission(perms, "k8s.list", "apps.v1.deployments", "list"))
	testza.AssertNotNil(t, plugin.CheckPermission(perms, "k8s.delete", "apps.v1.deployments", "delete"))
	testza.AssertNotNil(t, plugin.CheckPermission(perms, "k8s.list", "core.v1.pods", "list"))
	testza.AssertNil(t, plugin.CheckPermission(perms, "logs.stream", "", ""))
	testza.AssertNotNil(t, plugin.CheckPermission(perms, "exec.open", "", ""))
	testza.AssertNotNil(t, plugin.CheckPermission(perms, "unknown.method", "", ""))
}
