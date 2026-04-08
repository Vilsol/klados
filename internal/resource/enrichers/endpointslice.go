package enrichers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type EndpointSliceEnricher struct{}

func (e *EndpointSliceEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	labels, _, _ := unstructured.NestedStringMap(obj.Object, "metadata", "labels")
	serviceName := labels["kubernetes.io/service-name"]
	if serviceName == "" {
		ownerRefs, _, _ := unstructured.NestedSlice(obj.Object, "metadata", "ownerReferences")
		for _, ref := range ownerRefs {
			rm, ok := ref.(map[string]any)
			if !ok {
				continue
			}
			name, _ := rm["name"].(string)
			if name != "" {
				serviceName = name
				break
			}
		}
	}
	_ = unstructured.SetNestedField(obj.Object, serviceName, "status", "serviceDisplay")

	ports, _, _ := unstructured.NestedSlice(obj.Object, "ports")
	var portParts []string
	for _, p := range ports {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		name, _ := pm["name"].(string)
		protocol, _ := pm["protocol"].(string)
		port := int64(0)
		switch v := pm["port"].(type) {
		case int64:
			port = v
		case float64:
			port = int64(v)
		}
		if name != "" {
			portParts = append(portParts, fmt.Sprintf("%s:%d/%s", name, port, protocol))
		} else {
			portParts = append(portParts, fmt.Sprintf("%d/%s", port, protocol))
		}
	}
	_ = unstructured.SetNestedField(obj.Object, strings.Join(portParts, ", "), "status", "portsDisplay")

	endpoints, _, _ := unstructured.NestedSlice(obj.Object, "endpoints")
	_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d", len(endpoints)), "status", "endpointCount")

	return nil
}
