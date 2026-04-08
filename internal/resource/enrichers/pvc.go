package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type PVCEnricher struct{}

func (e *PVCEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	modes, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "accessModes")
	_ = unstructured.SetNestedField(obj.Object, abbreviateAccessModes(modes), "status", "accessModesDisplay")
	return nil
}
