package enrichers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type IngressEnricher struct{}

func (e *IngressEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	rules, _, _ := unstructured.NestedSlice(obj.Object, "spec", "rules")
	var hosts []string
	for _, r := range rules {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if h, _ := rm["host"].(string); h != "" {
			hosts = append(hosts, h)
		}
	}
	_ = unstructured.SetNestedField(obj.Object, strings.Join(hosts, ", "), "status", "hostsDisplay")

	svcName, _, _ := unstructured.NestedString(obj.Object, "spec", "defaultBackend", "service", "name")
	defaultBackendDisplay := ""
	if svcName != "" {
		port := int64(0)
		rawPort, _, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "defaultBackend", "service", "port", "number")
		switch v := rawPort.(type) {
		case int64:
			port = v
		case float64:
			port = int64(v)
		}
		defaultBackendDisplay = fmt.Sprintf("%s:%d", svcName, port)
	}
	_ = unstructured.SetNestedField(obj.Object, defaultBackendDisplay, "status", "defaultBackendDisplay")

	return nil
}
