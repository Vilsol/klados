package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestPriorityClassEnricher_GlobalDefaultDisplay_True(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"globalDefault": true,
	}}
	e := &enrichers.PriorityClassEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "globalDefaultDisplay")
	testza.AssertEqual(t, "Yes", display)
}

func TestPriorityClassEnricher_GlobalDefaultDisplay_False(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"globalDefault": false,
	}}
	e := &enrichers.PriorityClassEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "globalDefaultDisplay")
	testza.AssertEqual(t, "", display)
}

func TestPriorityClassEnricher_GlobalDefaultDisplay_Missing(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	e := &enrichers.PriorityClassEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "globalDefaultDisplay")
	testza.AssertEqual(t, "", display)
}
