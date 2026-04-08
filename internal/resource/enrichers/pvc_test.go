package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestPVCEnricher_AccessModesDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"accessModes": []any{"ReadWriteOnce"},
		},
	}}

	e := &enrichers.PVCEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "accessModesDisplay")
	testza.AssertEqual(t, "RWO", display)
}
