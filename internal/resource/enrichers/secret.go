package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type SecretEnricher struct{}

func (e *SecretEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	data, _, _ := unstructured.NestedMap(obj.Object, "data")
	_ = unstructured.SetNestedField(obj.Object, int64(len(data)), "status", "dataKeysCount")
	return nil
}
