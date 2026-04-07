package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

type mockDrainService struct {
	active map[string]bool
}

func (m *mockDrainService) IsActive(contextName, nodeName string) bool {
	return m.active[contextName+":"+nodeName]
}

func TestNodeEnricher_DrainPhase_SetWhenActive(t *testing.T) {
	svc := &mockDrainService{active: map[string]bool{"ctx:node1": true}}
	e := &enrichers.NodeEnricher{DrainService: svc}

	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "node1"},
		"status":   map[string]any{"conditions": []any{}},
		"spec":     map[string]any{},
	}}

	testza.AssertNoError(t, e.Enrich("ctx", obj))

	phase, _, _ := unstructured.NestedString(obj.Object, "status", "drainPhase")
	testza.AssertEqual(t, "Draining", phase)
}

func TestNodeEnricher_DrainPhase_AbsentWhenInactive(t *testing.T) {
	svc := &mockDrainService{active: map[string]bool{}}
	e := &enrichers.NodeEnricher{DrainService: svc}

	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "node1"},
		"status":   map[string]any{"conditions": []any{}},
		"spec":     map[string]any{},
	}}

	testza.AssertNoError(t, e.Enrich("ctx", obj))

	phase, exists, _ := unstructured.NestedString(obj.Object, "status", "drainPhase")
	testza.AssertFalse(t, exists)
	testza.AssertEqual(t, "", phase)
}

func TestNodeEnricher_DrainPhase_NilDrainService(t *testing.T) {
	e := &enrichers.NodeEnricher{}

	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "node1"},
		"status":   map[string]any{"conditions": []any{}},
		"spec":     map[string]any{},
	}}

	testza.AssertNoError(t, e.Enrich("ctx", obj))
}

func TestNodeEnricher_DrainPhase_ContextIsolation(t *testing.T) {
	svc := &mockDrainService{active: map[string]bool{"ctx1:node1": true}}
	e := &enrichers.NodeEnricher{DrainService: svc}

	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "node1"},
		"status":   map[string]any{"conditions": []any{}},
		"spec":     map[string]any{},
	}}

	// ctx2 should NOT have drain phase set even though ctx1:node1 is active
	testza.AssertNoError(t, e.Enrich("ctx2", obj))

	_, exists, _ := unstructured.NestedString(obj.Object, "status", "drainPhase")
	testza.AssertFalse(t, exists)
}
