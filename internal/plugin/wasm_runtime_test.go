package plugin_test

import (
	_ "embed"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/klados/internal/plugin"
)

//go:embed testdata/noop_enricher.wasm
var noopEnricherWasm []byte

func newTestRuntime(t *testing.T) *plugin.WasmRuntime {
	t.Helper()
	rt, err := plugin.NewWasmRuntime(t.Context(), noopEnricherWasm, "test-plugin", plugin.NewPermissionSet(nil), nil, plugin.HostAPIDeps{})
	testza.AssertNil(t, err)
	testza.AssertNotNil(t, rt)
	return rt
}

func TestWasmRuntime_LoadAndInit(t *testing.T) {
	rt := newTestRuntime(t)
	testza.AssertNil(t, rt.Close())
}

func TestWasmRuntime_CallEnrich_EchoesInput(t *testing.T) {
	rt := newTestRuntime(t)
	defer rt.Close()

	objJSON := []byte(`{"kind":"Pod","apiVersion":"v1"}`)
	result, err := rt.CallEnrich("core.v1.pods", objJSON)

	testza.AssertNil(t, err)
	testza.AssertNotNil(t, result)
	testza.AssertEqual(t, string(objJSON), string(result))
}

func TestWasmRuntime_CallEnrich_EmptyInput(t *testing.T) {
	rt := newTestRuntime(t)
	defer rt.Close()

	result, err := rt.CallEnrich("core.v1.pods", []byte{})
	testza.AssertNil(t, err)
	testza.AssertNil(t, result)
}

func TestWasmRuntime_Close_CallsDestroy(t *testing.T) {
	rt := newTestRuntime(t)
	// Close should not return an error even when called once.
	testza.AssertNil(t, rt.Close())
}
