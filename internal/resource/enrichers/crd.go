package enrichers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CRDEnricher struct{}

func (e *CRDEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	versions, _, _ := unstructured.NestedSlice(obj.Object, "spec", "versions")

	var versionNames []string
	var storageVersion string

	for _, v := range versions {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		name, _ := vm["name"].(string)
		if name == "" {
			continue
		}
		versionNames = append(versionNames, name)
		if storage, _ := vm["storage"].(bool); storage {
			storageVersion = name
		}
	}

	_ = unstructured.SetNestedField(obj.Object, strings.Join(versionNames, ", "), "status", "versionsDisplay")
	_ = unstructured.SetNestedField(obj.Object, storageVersion, "status", "storageVersion")

	if storageVersion != "" {
		group, _, _ := unstructured.NestedString(obj.Object, "spec", "group")
		plural, _, _ := unstructured.NestedString(obj.Object, "spec", "names", "plural")
		_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%s.%s.%s", group, storageVersion, plural), "status", "storageGVR")
	}

	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, c := range conditions {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if cm["type"] == "Established" {
			status, _ := cm["status"].(string)
			_ = unstructured.SetNestedField(obj.Object, status, "status", "established")
			break
		}
	}

	return nil
}
