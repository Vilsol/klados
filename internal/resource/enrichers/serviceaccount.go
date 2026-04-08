package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ServiceAccountEnricher struct{}

func (e *ServiceAccountEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	secrets, _, _ := unstructured.NestedSlice(obj.Object, "secrets")
	_ = unstructured.SetNestedField(obj.Object, int64(len(secrets)), "status", "secretsCount")
	return nil
}
