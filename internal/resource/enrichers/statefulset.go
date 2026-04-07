package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StatefulSetEnricher struct{}

func (e *StatefulSetEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	desired, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")

	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d/%d", ready, desired), "status", "readyDisplay")
	return nil
}
