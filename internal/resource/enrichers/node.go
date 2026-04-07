package enrichers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DrainStateProvider is a minimal interface so NodeEnricher doesn't import the services package.
type DrainStateProvider interface {
	IsActive(contextName, nodeName string) bool
}

type NodeEnricher struct {
	DrainService DrainStateProvider
}

func (e *NodeEnricher) Enrich(contextName string, obj *unstructured.Unstructured) error {
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")

	readyStatus := "Unknown"
	ready, total := 0, len(conditions)
	for _, c := range conditions {
		cMap, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if s, _ := cMap["status"].(string); s == "True" {
			ready++
		}
		if t, _ := cMap["type"].(string); t == "Ready" {
			if s, _ := cMap["status"].(string); s == "True" {
				readyStatus = "Ready"
			} else {
				readyStatus = "NotReady"
			}
		}
	}
	_ = unstructured.SetNestedField(obj.Object, readyStatus, "status", "readyStatus")
	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d/%d", ready, total), "status", "conditionsSummary")

	labels := obj.GetLabels()
	var roles []string
	for k := range labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			roles = append(roles, strings.TrimPrefix(k, "node-role.kubernetes.io/"))
		}
	}
	rolesStr := strings.Join(roles, ",")
	if rolesStr == "" {
		rolesStr = "<none>"
	}
	_ = unstructured.SetNestedField(obj.Object, rolesStr, "status", "roles")

	taints, _, _ := unstructured.NestedSlice(obj.Object, "spec", "taints")
	taintSummary := "<none>"
	if len(taints) > 0 {
		var effects []string
		for _, t := range taints {
			if m, ok := t.(map[string]any); ok {
				if e, _ := m["effect"].(string); e != "" {
					effects = append(effects, e)
				}
			}
		}
		taintSummary = fmt.Sprintf("%d (%s)", len(taints), strings.Join(effects, ", "))
	}
	_ = unstructured.SetNestedField(obj.Object, taintSummary, "status", "taintsSummary")

	allocatable, _, _ := unstructured.NestedStringMap(obj.Object, "status", "allocatable")
	if es, ok := allocatable["ephemeral-storage"]; ok {
		_ = unstructured.SetNestedField(obj.Object, es, "status", "ephemeralStorage")
	}

	if e.DrainService != nil && e.DrainService.IsActive(contextName, obj.GetName()) {
		_ = unstructured.SetNestedField(obj.Object, "Draining", "status", "drainPhase")
	}

	return nil
}
