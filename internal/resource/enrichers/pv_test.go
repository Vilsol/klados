package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestPVEnricher_AccessModesDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"accessModes": []any{"ReadWriteOnce", "ReadOnlyMany"},
		},
	}}

	e := &enrichers.PVEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "accessModesDisplay")
	testza.AssertEqual(t, "RWO,ROX", display)
}

func TestPVEnricher_ClaimDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"claimRef": map[string]any{
				"namespace": "default",
				"name":      "my-pvc",
			},
			"accessModes": []any{"ReadWriteOnce"},
		},
	}}

	e := &enrichers.PVEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "claimDisplay")
	testza.AssertEqual(t, "default/my-pvc", display)
}

func TestPVEnricher_NoClaimRef(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"accessModes": []any{"ReadWriteOnce"},
		},
	}}

	e := &enrichers.PVEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "claimDisplay")
	testza.AssertEqual(t, "", display)
}
