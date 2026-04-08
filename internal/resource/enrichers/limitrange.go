package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type LimitRangeEnricher struct{}

func (e *LimitRangeEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	limits, _, _ := unstructured.NestedSlice(obj.Object, "spec", "limits")
	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d", len(limits)), "status", "limitCount")
	return nil
}
