package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestBindingEnricher_RoleRefDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"roleRef": map[string]any{
			"kind": "ClusterRole",
			"name": "admin",
		},
	}}

	e := &enrichers.BindingEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "roleRefDisplay")
	testza.AssertEqual(t, "ClusterRole/admin", display)
}

func TestBindingEnricher_SubjectsCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"roleRef": map[string]any{"kind": "Role", "name": "reader"},
		"subjects": []any{
			map[string]any{"kind": "User", "name": "alice"},
			map[string]any{"kind": "User", "name": "bob"},
		},
	}}

	e := &enrichers.BindingEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "subjectsCount")
	testza.AssertEqual(t, int64(2), count)
}
