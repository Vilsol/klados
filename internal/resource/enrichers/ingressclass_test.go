package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestIngressClassEnricher_IsDefault_True(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				"ingressclass.kubernetes.io/is-default-class": "true",
			},
		},
	}}
	e := &enrichers.IngressClassEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "isDefault")
	testza.AssertEqual(t, "Yes", display)
}

func TestIngressClassEnricher_IsDefault_False(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				"ingressclass.kubernetes.io/is-default-class": "false",
			},
		},
	}}
	e := &enrichers.IngressClassEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "isDefault")
	testza.AssertEqual(t, "", display)
}

func TestIngressClassEnricher_IsDefault_Missing(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{},
	}}
	e := &enrichers.IngressClassEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "isDefault")
	testza.AssertEqual(t, "", display)
}
