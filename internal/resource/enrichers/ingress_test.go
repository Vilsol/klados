package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestIngressEnricher_HostsDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"rules": []any{
				map[string]any{"host": "foo.com"},
				map[string]any{"host": "bar.com"},
			},
		},
	}}

	e := &enrichers.IngressEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "hostsDisplay")
	testza.AssertEqual(t, "foo.com, bar.com", display)
}

func TestIngressEnricher_DefaultBackendDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"defaultBackend": map[string]any{
				"service": map[string]any{
					"name": "my-service",
					"port": map[string]any{"number": int64(8080)},
				},
			},
		},
	}}

	e := &enrichers.IngressEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "defaultBackendDisplay")
	testza.AssertEqual(t, "my-service:8080", display)
}

func TestIngressEnricher_NoDefaultBackend(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{},
	}}

	e := &enrichers.IngressEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "defaultBackendDisplay")
	testza.AssertEqual(t, "", display)
}
