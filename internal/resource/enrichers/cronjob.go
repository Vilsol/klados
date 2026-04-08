package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CronJobEnricher struct{}

func (e *CronJobEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	active, _, _ := unstructured.NestedSlice(obj.Object, "status", "active")
	_ = unstructured.SetNestedField(obj.Object, int64(len(active)), "status", "activeCount")
	return nil
}
