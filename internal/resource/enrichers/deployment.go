package enrichers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type DeploymentEnricher struct{}

func (e *DeploymentEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	desired, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	available, _, _ := unstructured.NestedInt64(obj.Object, "status", "availableReplicas")

	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d/%d", ready, desired), "status", "readyDisplay")
	_ = unstructured.SetNestedField(obj.Object, available, "status", "availableReplicas")
	return nil
}
