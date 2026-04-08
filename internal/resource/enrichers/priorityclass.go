package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type PriorityClassEnricher struct{}

func (e *PriorityClassEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	globalDefault, found, _ := unstructured.NestedBool(obj.Object, "globalDefault")
	if found && globalDefault {
		_ = unstructured.SetNestedField(obj.Object, "Yes", "status", "globalDefaultDisplay")
	} else {
		_ = unstructured.SetNestedField(obj.Object, "", "status", "globalDefaultDisplay")
	}
	return nil
}
