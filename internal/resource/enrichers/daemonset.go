package enrichers

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type DaemonSetEnricher struct{}

func (e *DaemonSetEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "numberReady")
	desired, _, _ := unstructured.NestedInt64(obj.Object, "status", "desiredNumberScheduled")

	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d/%d", ready, desired), "status", "readyDisplay")

	nodeSelector, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "nodeSelector")
	if len(nodeSelector) > 0 {
		keys := make([]string, 0, len(nodeSelector))
		for k := range nodeSelector {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+"="+nodeSelector[k])
		}
		_ = unstructured.SetNestedField(obj.Object, strings.Join(parts, ","), "status", "nodeSelectorDisplay")
	} else {
		_ = unstructured.SetNestedField(obj.Object, "", "status", "nodeSelectorDisplay")
	}

	return nil
}
