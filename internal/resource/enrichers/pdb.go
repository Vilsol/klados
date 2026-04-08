package enrichers

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type PDBEnricher struct{}

func (e *PDBEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	matchLabels, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector", "matchLabels")
	if len(matchLabels) == 0 {
		_ = unstructured.SetNestedField(obj.Object, "<all pods>", "status", "podSelectorDisplay")
	} else {
		keys := make([]string, 0, len(matchLabels))
		for k := range matchLabels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", k, matchLabels[k]))
		}
		_ = unstructured.SetNestedField(obj.Object, strings.Join(parts, ", "), "status", "podSelectorDisplay")
	}
	return nil
}
