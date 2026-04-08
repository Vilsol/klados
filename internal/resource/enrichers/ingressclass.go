package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type IngressClassEnricher struct{}

func (e *IngressClassEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	annotations, _, _ := unstructured.NestedStringMap(obj.Object, "metadata", "annotations")
	if annotations["ingressclass.kubernetes.io/is-default-class"] == "true" {
		_ = unstructured.SetNestedField(obj.Object, "Yes", "status", "isDefault")
	} else {
		_ = unstructured.SetNestedField(obj.Object, "", "status", "isDefault")
	}
	return nil
}
