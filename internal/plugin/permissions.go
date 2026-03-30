package plugin

import (
	"fmt"

	"github.com/Vilsol/klados/internal/plugin/types"
	"github.com/Vilsol/klados/internal/resource"
)

type PermissionSet struct {
	p *types.Permissions
}

func NewPermissionSet(p *types.Permissions) PermissionSet {
	return PermissionSet{p: p}
}

func (ps PermissionSet) AllowsLogs() bool {
	return ps.p != nil && ps.p.Logs != nil && *ps.p.Logs
}

func (ps PermissionSet) AllowsExec() bool {
	return ps.p != nil && ps.p.Exec != nil && *ps.p.Exec
}

func (ps PermissionSet) AllowsEvents() bool {
	return ps.p != nil && ps.p.Events != nil && *ps.p.Events
}

func (ps PermissionSet) AllowsStorage() bool {
	return ps.p != nil && ps.p.Storage != nil && *ps.p.Storage
}

func (ps PermissionSet) AllowsWasi(cap string) bool {
	if ps.p == nil || ps.p.Wasi == nil {
		return false
	}
	w := ps.p.Wasi
	switch cap {
	case "clock":
		return w.Clock != nil && *w.Clock
	case "env":
		return w.Env != nil && *w.Env
	case "filesystem":
		return w.Filesystem != nil && *w.Filesystem
	case "network":
		return w.Network != nil && *w.Network
	}
	return false
}

// AllowsResource checks whether the manifest permits the given GVR + verb.
// gvr is in dot-separated format (e.g. "apps.v1.deployments").
func (ps PermissionSet) AllowsResource(gvr, verb string) bool {
	if ps.p == nil {
		return false
	}
	parsed, err := resource.ParseGVR(gvr)
	if err != nil {
		return false
	}
	for _, rp := range ps.p.Resources {
		if rp.Group == parsed.Group && rp.Version == parsed.Version && rp.Resource == parsed.Resource {
			for _, v := range rp.Verbs {
				if string(v) == verb {
					return true
				}
			}
		}
	}
	return false
}

// CheckPermission returns a structured error if the method+gvr+verb is denied.
// method is the host_call method name (e.g. "k8s.list").
func CheckPermission(perms PermissionSet, method, gvr, verb string) error {
	switch {
	case method == "logs.stream":
		if !perms.AllowsLogs() {
			return fmt.Errorf("method not available: %s", method)
		}
	case method == "exec.open":
		if !perms.AllowsExec() {
			return fmt.Errorf("method not available: %s", method)
		}
	case method == "event.subscribe":
		if !perms.AllowsEvents() {
			return fmt.Errorf("method not available: %s", method)
		}
	case method == "storage.get" || method == "storage.set" || method == "storage.delete":
		if !perms.AllowsStorage() {
			return fmt.Errorf("method not available: %s", method)
		}
	case method == "k8s.list" || method == "k8s.get" || method == "k8s.create" ||
		method == "k8s.update" || method == "k8s.delete" || method == "k8s.watch":
		if !perms.AllowsResource(gvr, verb) {
			return fmt.Errorf("permission denied: %s on %s", verb, gvr)
		}
	default:
		return fmt.Errorf("method not available: %s", method)
	}
	return nil
}
