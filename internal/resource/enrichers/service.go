package enrichers

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ServiceEnricher struct{}

func (e *ServiceEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	ports, _, _ := unstructured.NestedSlice(obj.Object, "spec", "ports")
	var portParts []string
	for _, p := range ports {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		port := int64(0)
		switch v := pm["port"].(type) {
		case int64:
			port = v
		case float64:
			port = int64(v)
		}
		protocol, _ := pm["protocol"].(string)
		if protocol == "" {
			protocol = "TCP"
		}
		nodePort := int64(0)
		switch v := pm["nodePort"].(type) {
		case int64:
			nodePort = v
		case float64:
			nodePort = int64(v)
		}
		if nodePort != 0 {
			portParts = append(portParts, fmt.Sprintf("%d:%d/%s", port, nodePort, protocol))
		} else {
			portParts = append(portParts, fmt.Sprintf("%d/%s", port, protocol))
		}
	}
	_ = unstructured.SetNestedField(obj.Object, strings.Join(portParts, ", "), "status", "portsDisplay")

	var externalIPs []string
	ingresses, _, _ := unstructured.NestedSlice(obj.Object, "status", "loadBalancer", "ingress")
	for _, ing := range ingresses {
		im, ok := ing.(map[string]any)
		if !ok {
			continue
		}
		if ip, _ := im["ip"].(string); ip != "" {
			externalIPs = append(externalIPs, ip)
		}
	}
	if len(externalIPs) == 0 {
		specIPs, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "externalIPs")
		externalIPs = specIPs
	}
	_ = unstructured.SetNestedField(obj.Object, strings.Join(externalIPs, ", "), "status", "externalIPDisplay")

	return nil
}
