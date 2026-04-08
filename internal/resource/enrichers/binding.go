package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type BindingEnricher struct{}

func (e *BindingEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	kind, _, _ := unstructured.NestedString(obj.Object, "roleRef", "kind")
	name, _, _ := unstructured.NestedString(obj.Object, "roleRef", "name")
	_ = unstructured.SetNestedField(obj.Object, kind+"/"+name, "status", "roleRefDisplay")

	subjects, _, _ := unstructured.NestedSlice(obj.Object, "subjects")
	_ = unstructured.SetNestedField(obj.Object, int64(len(subjects)), "status", "subjectsCount")

	return nil
}
