package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestServiceEnricher_PortsDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"ports": []any{
				map[string]any{"port": int64(80), "protocol": "TCP"},
				map[string]any{"port": int64(443), "protocol": "TCP"},
			},
		},
	}}

	e := &enrichers.ServiceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "portsDisplay")
	testza.AssertEqual(t, "80/TCP, 443/TCP", display)
}

func TestServiceEnricher_PortsDisplay_WithNodePort(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"ports": []any{
				map[string]any{"port": int64(80), "protocol": "TCP", "nodePort": int64(30080)},
			},
		},
	}}

	e := &enrichers.ServiceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "portsDisplay")
	testza.AssertEqual(t, "80:30080/TCP", display)
}

func TestServiceEnricher_ExternalIPDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"loadBalancer": map[string]any{
				"ingress": []any{
					map[string]any{"ip": "1.2.3.4"},
				},
			},
		},
	}}

	e := &enrichers.ServiceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "externalIPDisplay")
	testza.AssertEqual(t, "1.2.3.4", display)
}
