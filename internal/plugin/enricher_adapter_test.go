package plugin_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/plugin"
)

// fakeRuntime implements plugin.EnrichRuntime for unit tests.
type fakeRuntime struct {
	result []byte
	err    error
}

func (f *fakeRuntime) CallEnrich(_ string, _ []byte) ([]byte, error) {
	return f.result, f.err
}

func newEnricher(rt plugin.EnrichRuntime) *plugin.PluginEnricher {
	return &plugin.PluginEnricher{Runtime: rt, GVR: "core.v1.pods", PluginName: "test"}
}

func TestPluginEnricher_NewFields_AreMerged(t *testing.T) {
	result, _ := json.Marshal(map[string]any{"kind": "Pod", "pluginField": "added"})
	e := newEnricher(&fakeRuntime{result: result})

	obj := &unstructured.Unstructured{Object: map[string]any{"kind": "Pod"}}
	testza.AssertNil(t, e.Enrich(obj))

	testza.AssertEqual(t, "added", obj.Object["pluginField"])
}

func TestPluginEnricher_CollisionField_StillMerged(t *testing.T) {
	result, _ := json.Marshal(map[string]any{"kind": "OverwrittenKind", "newField": "yes"})
	e := newEnricher(&fakeRuntime{result: result})

	obj := &unstructured.Unstructured{Object: map[string]any{"kind": "Pod"}}
	testza.AssertNil(t, e.Enrich(obj))

	// Collision field is merged (warn but don't suppress).
	testza.AssertEqual(t, "OverwrittenKind", obj.Object["kind"])
	testza.AssertEqual(t, "yes", obj.Object["newField"])
}

func TestPluginEnricher_RuntimeError_ObjectUnchanged(t *testing.T) {
	e := newEnricher(&fakeRuntime{err: errors.New("wasm trap")})

	obj := &unstructured.Unstructured{Object: map[string]any{"kind": "Pod"}}
	testza.AssertNil(t, e.Enrich(obj))
	testza.AssertEqual(t, "Pod", obj.Object["kind"])
	testza.AssertNil(t, obj.Object["pluginField"])
}

func TestPluginEnricher_EmptyResult_ObjectUnchanged(t *testing.T) {
	e := newEnricher(&fakeRuntime{result: nil})

	obj := &unstructured.Unstructured{Object: map[string]any{"kind": "Pod"}}
	testza.AssertNil(t, e.Enrich(obj))
	testza.AssertEqual(t, "Pod", obj.Object["kind"])
}

func TestPluginEnricher_WithRealRuntime_EchoesInput(t *testing.T) {
	rt := newTestRuntime(t)
	defer rt.Close()

	e := &plugin.PluginEnricher{Runtime: rt, GVR: "core.v1.pods", PluginName: "test"}
	obj := &unstructured.Unstructured{Object: map[string]any{"kind": "Pod", "apiVersion": "v1"}}

	testza.AssertNil(t, e.Enrich(obj))
	testza.AssertEqual(t, "Pod", obj.Object["kind"])
	testza.AssertEqual(t, "v1", obj.Object["apiVersion"])
}
