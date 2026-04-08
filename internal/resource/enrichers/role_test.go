package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestRoleEnricher_RulesCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"rules": []any{
			map[string]any{"verbs": []any{"get"}},
			map[string]any{"verbs": []any{"list"}},
			map[string]any{"verbs": []any{"watch"}},
		},
	}}

	e := &enrichers.RoleEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "rulesCount")
	testza.AssertEqual(t, int64(3), count)
}

func TestRoleEnricher_NoRules(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{}}

	e := &enrichers.RoleEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "rulesCount")
	testza.AssertEqual(t, int64(0), count)
}
