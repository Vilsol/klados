//go:build wasip1

package main

import (
	"fmt"

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

	// Taint report: aggregate taint keys across all nodes and save to storage.
	sdk.OnCommand("node-annotator-taint-report", func() {
		nodes, err := sdk.K8s.List("core.v1.nodes", "")
		if err != nil {
			sdk.Log.Error("taint report: " + err.Error())
			return
		}

		// taintKey → set of node names
		keyNodes := map[string][]string{}
		for _, node := range nodes {
			meta, _ := node["metadata"].(map[string]any)
			nodeName, _ := meta["name"].(string)
			spec, _ := node["spec"].(map[string]any)
			taints, _ := spec["taints"].([]any)
			for _, t := range taints {
				taint, ok := t.(map[string]any)
				if !ok {
					continue
				}
				key, _ := taint["key"].(string)
				keyNodes[key] = append(keyNodes[key], nodeName)
			}
		}

		sdk.Log.Info(fmt.Sprintf("taint report: %d node(s), %d unique taint key(s)", len(nodes), len(keyNodes)))
		for key, affected := range keyNodes {
			sdk.Log.Info(fmt.Sprintf("  %s → %d node(s): %v", key, len(affected), affected))
		}

		// Persist last-run summary to storage so it can be read by other SDK calls.
		summary := fmt.Sprintf("nodes:%d taint-keys:%d", len(nodes), len(keyNodes))
		_ = sdk.Storage.Set("last-taint-report", summary)
		sdk.Log.Info("taint report: saved to storage (key=last-taint-report)")
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
