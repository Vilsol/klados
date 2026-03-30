//go:build wasip1

package main

import (
	sdk "github.com/Vilsol/klados-plugin-sdk"
)

func init() {
	// Store default annotation prefix (demonstrates storage write)
	sdk.Storage.Set("annotation-prefix", "klados.io")

	// Subscribe to cluster:connected events (demonstrates event subscription)
	sdk.OnEvent("cluster:connected", func(payload []byte) {
		prefix, _, _ := sdk.Storage.Get("annotation-prefix")
		sdk.Log.Info("cluster connected, using annotation prefix: " + prefix)
	})

	sdk.RegisterEnricher("core.v1.nodes", enrichNode)
}

func enrichNode(obj map[string]any) map[string]any {
	// Read annotation prefix from storage (demonstrates storage read in hot path)
	prefix, found, _ := sdk.Storage.Get("annotation-prefix")
	if !found || prefix == "" {
		prefix = "klados.io"
	}

	status, _ := obj["status"].(map[string]any)
	if status == nil {
		status = map[string]any{}
	}

	spec, _ := obj["spec"].(map[string]any)
	taintCount := 0
	if spec != nil {
		taints, _ := spec["taints"].([]any)
		taintCount = len(taints)
	}

	conditions, _ := status["conditions"].([]any)
	readySummary := "Unknown"
	for _, c := range conditions {
		cond, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if cond["type"] == "Ready" {
			if cond["status"] == "True" {
				readySummary = "Ready"
			} else {
				readySummary = "NotReady"
				if reason, ok := cond["reason"].(string); ok && reason != "" {
					readySummary = "NotReady:" + reason
				}
			}
			break
		}
	}

	return map[string]any{
		"status": map[string]any{
			"taintCount":       taintCount,
			"readinessSummary": readySummary,
			"annotationPrefix": prefix,
		},
	}
}

func main() {}
