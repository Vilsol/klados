package enrichers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StorageClassEnricher struct{}

func (e *StorageClassEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	isDefault := "false"
	if obj.GetAnnotations()["storageclass.kubernetes.io/is-default-class"] == "true" {
		isDefault = "true"
	}
	_ = unstructured.SetNestedField(obj.Object, isDefault, "status", "isDefault")
	return nil
}
