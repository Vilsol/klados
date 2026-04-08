package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ResourceQuotaEnricher struct{}

func (e *ResourceQuotaEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	hard, _, _ := unstructured.NestedMap(obj.Object, "spec", "hard")
	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d", len(hard)), "status", "resourceCount")
	return nil
}
