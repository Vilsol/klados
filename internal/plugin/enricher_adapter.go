package plugin

import (
	"context"
	"encoding/json"

	"github.com/Vilsol/slox"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EnrichRuntime is the subset of WasmRuntime used by PluginEnricher.
type EnrichRuntime interface {
	CallEnrich(gvr string, objJSON []byte) ([]byte, error)
}

// PluginEnricher adapts a WasmRuntime to the resource.Enricher interface.
type PluginEnricher struct {
	Runtime    EnrichRuntime
	GVR        string
	PluginName string
	Ctx        context.Context
	OnError    func(error) // called when CallEnrich fails (e.g. Wasm trap); optional
}

func (e *PluginEnricher) GetPluginName() string {
	return e.PluginName
}

func (e *PluginEnricher) Enrich(obj *unstructured.Unstructured) error {
	ctx := e.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	objJSON, err := json.Marshal(obj.Object)
	if err != nil {
		slox.Warn(ctx, "plugin enricher: marshal failed",
			"plugin", e.PluginName, "gvr", e.GVR, "error", err)
		return nil
	}

	resultJSON, err := e.Runtime.CallEnrich(e.GVR, objJSON)
	if err != nil {
		slox.Warn(ctx, "plugin enricher: call failed",
			"plugin", e.PluginName, "gvr", e.GVR, "error", err)
		if e.OnError != nil {
			e.OnError(err)
		}
		return nil
	}
	if len(resultJSON) == 0 {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		slox.Warn(ctx, "plugin enricher: unmarshal failed",
			"plugin", e.PluginName, "gvr", e.GVR, "error", err)
		return nil
	}

	deepMerge(ctx, e.PluginName, e.GVR, obj.Object, result)

	return nil
}

// deepMerge recursively merges src into dst.
// When both values are maps, it recurses rather than replacing.
// Leaf value overwrites (string/number/bool/array) emit a warning.
func deepMerge(ctx context.Context, plugin, gvr string, dst, src map[string]any) {
	for k, v := range src {
		existing, exists := dst[k]
		if !exists {
			dst[k] = v
			continue
		}
		existingMap, existingIsMap := existing.(map[string]any)
		srcMap, srcIsMap := v.(map[string]any)
		if existingIsMap && srcIsMap {
			deepMerge(ctx, plugin, gvr, existingMap, srcMap)
		} else {
			slox.Warn(ctx, "plugin enricher overwrote existing field",
				"plugin", plugin, "gvr", gvr, "field", k)
			dst[k] = v
		}
	}
}
