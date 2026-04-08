package enrichers

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type NetworkPolicyEnricher struct{}

func (e *NetworkPolicyEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	matchLabels, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "podSelector", "matchLabels")
	if len(matchLabels) == 0 {
		_ = unstructured.SetNestedField(obj.Object, "<all pods>", "status", "podSelectorDisplay")
	} else {
		keys := make([]string, 0, len(matchLabels))
		for k := range matchLabels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", k, matchLabels[k]))
		}
		_ = unstructured.SetNestedField(obj.Object, strings.Join(parts, ", "), "status", "podSelectorDisplay")
	}

	policyTypes, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "policyTypes")
	_ = unstructured.SetNestedField(obj.Object, strings.Join(policyTypes, ", "), "status", "policyTypesDisplay")

	_, ingressFound, _ := unstructured.NestedSlice(obj.Object, "spec", "ingress")
	if !ingressFound {
		_ = unstructured.SetNestedField(obj.Object, "-", "status", "ingressRuleCount")
	} else {
		ingress, _, _ := unstructured.NestedSlice(obj.Object, "spec", "ingress")
		_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d", len(ingress)), "status", "ingressRuleCount")
	}

	_, egressFound, _ := unstructured.NestedSlice(obj.Object, "spec", "egress")
	if !egressFound {
		_ = unstructured.SetNestedField(obj.Object, "-", "status", "egressRuleCount")
	} else {
		egress, _, _ := unstructured.NestedSlice(obj.Object, "spec", "egress")
		_ = unstructured.SetNestedField(obj.Object, fmt.Sprintf("%d", len(egress)), "status", "egressRuleCount")
	}

	return nil
}
