package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type RoleEnricher struct{}

func (e *RoleEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	rules, _, _ := unstructured.NestedSlice(obj.Object, "rules")
	_ = unstructured.SetNestedField(obj.Object, int64(len(rules)), "status", "rulesCount")
	return nil
}
