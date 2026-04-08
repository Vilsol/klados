package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestPDBEnricher_PodSelectorDisplay_WithLabels(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"selector": map[string]any{
				"matchLabels": map[string]any{
					"app": "myapp",
				},
			},
		},
	}}
	e := &enrichers.PDBEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "podSelectorDisplay")
	testza.AssertEqual(t, "app=myapp", display)
}

func TestPDBEnricher_PodSelectorDisplay_Empty(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"selector": map[string]any{
				"matchLabels": map[string]any{},
			},
		},
	}}
	e := &enrichers.PDBEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "podSelectorDisplay")
	testza.AssertEqual(t, "<all pods>", display)
}
