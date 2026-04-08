package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type LeaseEnricher struct{}

func (e *LeaseEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	seconds, found, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "leaseDurationSeconds")
	if !found || seconds == nil {
		_ = unstructured.SetNestedField(obj.Object, "", "status", "leaseDurationDisplay")
		return nil
	}

	var secs int64
	switch v := seconds.(type) {
	case int64:
		secs = v
	case float64:
		secs = int64(v)
	default:
		_ = unstructured.SetNestedField(obj.Object, "", "status", "leaseDurationDisplay")
		return nil
	}

	var display string
	if secs < 60 {
		display = fmt.Sprintf("%ds", secs)
	} else {
		display = fmt.Sprintf("%dm", secs/60)
	}
	_ = unstructured.SetNestedField(obj.Object, display, "status", "leaseDurationDisplay")
	return nil
}
