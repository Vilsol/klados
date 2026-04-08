package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestNetworkPolicyEnricher_PodSelectorDisplay_Empty(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"podSelector": map[string]any{},
		},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "podSelectorDisplay")
	testza.AssertEqual(t, "<all pods>", display)
}

func TestNetworkPolicyEnricher_PodSelectorDisplay_WithLabels(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"podSelector": map[string]any{
				"matchLabels": map[string]any{
					"app": "nginx",
					"env": "prod",
				},
			},
		},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "podSelectorDisplay")
	testza.AssertEqual(t, "app=nginx, env=prod", display)
}

func TestNetworkPolicyEnricher_IngressRuleCount_Nil(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "ingressRuleCount")
	testza.AssertEqual(t, "-", count)
}

func TestNetworkPolicyEnricher_IngressRuleCount_Empty(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"ingress": []any{},
		},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "ingressRuleCount")
	testza.AssertEqual(t, "0", count)
}

func TestNetworkPolicyEnricher_IngressRuleCount_WithRules(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"ingress": []any{
				map[string]any{},
				map[string]any{},
			},
		},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "ingressRuleCount")
	testza.AssertEqual(t, "2", count)
}

func TestNetworkPolicyEnricher_EgressRuleCount_Nil(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "egressRuleCount")
	testza.AssertEqual(t, "-", count)
}

func TestNetworkPolicyEnricher_EgressRuleCount_Empty(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"egress": []any{},
		},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "egressRuleCount")
	testza.AssertEqual(t, "0", count)
}

func TestNetworkPolicyEnricher_PolicyTypesDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"policyTypes": []any{"Ingress", "Egress"},
		},
	}}
	e := &enrichers.NetworkPolicyEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "policyTypesDisplay")
	testza.AssertEqual(t, "Ingress, Egress", display)
}
