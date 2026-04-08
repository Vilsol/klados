package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestEndpointSliceEnricher_ServiceDisplay_FromLabel(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				"kubernetes.io/service-name": "my-service",
			},
		},
	}}
	e := &enrichers.EndpointSliceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "serviceDisplay")
	testza.AssertEqual(t, "my-service", display)
}

func TestEndpointSliceEnricher_ServiceDisplay_FallbackOwnerRef(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"ownerReferences": []any{
				map[string]any{
					"kind": "Service",
					"name": "owner-service",
				},
			},
		},
	}}
	e := &enrichers.EndpointSliceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "serviceDisplay")
	testza.AssertEqual(t, "owner-service", display)
}

func TestEndpointSliceEnricher_PortsDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"ports": []any{
			map[string]any{"name": "http", "port": int64(80), "protocol": "TCP"},
			map[string]any{"name": "https", "port": int64(443), "protocol": "TCP"},
		},
	}}
	e := &enrichers.EndpointSliceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "portsDisplay")
	testza.AssertEqual(t, "http:80/TCP, https:443/TCP", display)
}

func TestEndpointSliceEnricher_EndpointCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"endpoints": []any{
			map[string]any{"addresses": []any{"10.0.0.1"}},
			map[string]any{"addresses": []any{"10.0.0.2"}},
			map[string]any{"addresses": []any{"10.0.0.3"}},
		},
	}}
	e := &enrichers.EndpointSliceEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "endpointCount")
	testza.AssertEqual(t, "3", count)
}
