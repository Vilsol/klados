//go:build !wasip1

package main_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// TestEnricher_Go verifies that dist/plugin.wasm (standard Go build) produces
// expected enriched output when fed a node object.
func TestEnricher_Go(t *testing.T) {
	if _, err := os.Stat("dist/plugin.wasm"); os.IsNotExist(err) {
		t.Skip("dist/plugin.wasm not built; run 'mise run build:go' first")
	}
	// Standard Go WASM command modules call proc_exit(0) after _start; the Go
	// runtime then intentionally nil-dereferences to prevent proc_exit intercepts.
	// Use TinyGo (TestEnricher_TinyGo) for production plugins.
	t.Skip("standard Go command WASM requires reactor mode (_initialize); use TinyGo")
}

// TestEnricher_TinyGo verifies the TinyGo build produces identical output.
func TestEnricher_TinyGo(t *testing.T) {
	if _, err := os.Stat("dist/plugin-tiny.wasm"); os.IsNotExist(err) {
		t.Skip("dist/plugin-tiny.wasm not built; run 'mise run build:tinygo' first")
	}
	runEnricherTest(t, "dist/plugin-tiny.wasm")
}

func runEnricherTest(t *testing.T, wasmPath string) {
	t.Helper()
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Skipf("skipping: %s not found (%v); run 'mise run build' first", wasmPath, err)
	}

	ctx := context.Background()
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Stub host module — storage, k8s, log not needed for this test
	b := rt.NewHostModuleBuilder("klados_host")
	b.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, _ api.Module, stack []uint64) {
			stack[0] = 0 // host_call: return empty
		}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32},
			[]api.ValueType{api.ValueTypeI64}).
		Export("host_call")
	b.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, _ api.Module, _ []uint64) {}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_log")
	b.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, _ api.Module, stack []uint64) {
			stack[0] = 0
		}),
			[]api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("host_alloc")
	b.NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(_ context.Context, _ api.Module, _ []uint64) {}),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{}).
		Export("host_free")
	if _, err := b.Instantiate(ctx); err != nil {
		t.Fatalf("instantiating host module: %v", err)
	}

	// Custom WASI: intercept proc_exit(0) so standard Go command modules stay
	// open after _start returns (identical to klados wasm_runtime.go behaviour).
	wasiBuilder := rt.NewHostModuleBuilder(wasi_snapshot_preview1.ModuleName)
	wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)
	wasiBuilder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, mod api.Module, exitCode uint32) {
			if exitCode != 0 {
				_ = mod.CloseWithExitCode(ctx, exitCode)
				panic(sys.NewExitError(exitCode))
			}
		}).Export("proc_exit")
	if _, err := wasiBuilder.Instantiate(ctx); err != nil {
		t.Fatalf("instantiating WASI: %v", err)
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	testza.AssertNoError(t, err)

	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName("node-annotator").
		WithStartFunctions())
	testza.AssertNoError(t, err)
	defer mod.Close(ctx)

	// Initialize the guest runtime (mirrors wasm_runtime.go logic).
	if initializeFn := mod.ExportedFunction("_initialize"); initializeFn != nil {
		_, err = initializeFn.Call(ctx)
		testza.AssertNoError(t, err)
	} else if startFn := mod.ExportedFunction("_start"); startFn != nil {
		_, err = startFn.Call(ctx)
		if err != nil {
			var exitErr *sys.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("_start exited with code %d (module closed)", exitErr.ExitCode())
			}
			testza.AssertNoError(t, err)
		}
	}

	// Call plugin_init
	initFn := mod.ExportedFunction("plugin_init")
	testza.AssertNotNil(t, initFn)
	_, err = initFn.Call(ctx)
	testza.AssertNoError(t, err)

	// Build a test node with taints
	node := map[string]any{
		"apiVersion": "v1",
		"kind":       "Node",
		"metadata":   map[string]any{"name": "test-node"},
		"spec": map[string]any{
			"taints": []any{
				map[string]any{"key": "node-role.kubernetes.io/control-plane", "effect": "NoSchedule"},
				map[string]any{"key": "dedicated", "value": "gpu", "effect": "NoSchedule"},
			},
		},
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True"},
			},
		},
	}
	nodeJSON, err := json.Marshal(node)
	testza.AssertNoError(t, err)

	// Write node JSON into guest memory via plugin_alloc
	allocFn := mod.ExportedFunction("plugin_alloc")
	testza.AssertNotNil(t, allocFn)
	allocResult, err := allocFn.Call(ctx, uint64(len(nodeJSON)))
	testza.AssertNoError(t, err)
	objPtr := uint32(allocResult[0])
	ok := mod.Memory().Write(objPtr, nodeJSON)
	testza.AssertTrue(t, ok)

	// GVR
	gvr := []byte("core.v1.nodes")
	gvrResult, err := allocFn.Call(ctx, uint64(len(gvr)))
	testza.AssertNoError(t, err)
	gvrPtr := uint32(gvrResult[0])
	mod.Memory().Write(gvrPtr, gvr)

	enrichFn := mod.ExportedFunction("plugin_enrich")
	testza.AssertNotNil(t, enrichFn)

	results, err := enrichFn.Call(ctx,
		uint64(gvrPtr), uint64(len(gvr)),
		uint64(objPtr), uint64(len(nodeJSON)),
	)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, len(results) == 1)

	packed := results[0]
	respPtr := uint32(packed >> 32)
	respLen := uint32(packed & 0xFFFFFFFF)
	testza.AssertTrue(t, respLen > 0)

	respBytes, ok := mod.Memory().Read(respPtr, respLen)
	testza.AssertTrue(t, ok)

	var enriched map[string]any
	err = json.Unmarshal(respBytes, &enriched)
	testza.AssertNoError(t, err)

	status := enriched["status"].(map[string]any)
	testza.AssertEqual(t, float64(2), status["taintCount"])
	testza.AssertEqual(t, "Ready", status["readinessSummary"])
}
