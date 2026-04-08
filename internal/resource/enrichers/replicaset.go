package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ReplicaSetEnricher struct{}

func (e *ReplicaSetEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	owners, _, _ := unstructured.NestedSlice(obj.Object, "metadata", "ownerReferences")
	ownerDisplay := "<none>"
	if len(owners) > 0 {
		if m, ok := owners[0].(map[string]any); ok {
			if name, _ := m["name"].(string); name != "" {
				ownerDisplay = name
			}
		}
	}
	_ = unstructured.SetNestedField(obj.Object, ownerDisplay, "status", "ownerDisplay")
	return nil
}
