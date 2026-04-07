package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type DaemonSetEnricher struct{}

func (e *DaemonSetEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "numberReady")
	desired, _, _ := unstructured.NestedInt64(obj.Object, "status", "desiredNumberScheduled")

	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d/%d", ready, desired), "status", "readyDisplay")
	return nil
}
