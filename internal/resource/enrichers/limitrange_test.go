package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestLimitRangeEnricher_LimitCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"limits": []any{
				map[string]any{"type": "Container"},
				map[string]any{"type": "Pod"},
				map[string]any{"type": "PersistentVolumeClaim"},
			},
		},
	}}
	e := &enrichers.LimitRangeEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "limitCount")
	testza.AssertEqual(t, "3", count)
}

func TestLimitRangeEnricher_LimitCount_Empty(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"limits": []any{},
		},
	}}
	e := &enrichers.LimitRangeEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	count, _, _ := unstructured.NestedString(obj.Object, "status", "limitCount")
	testza.AssertEqual(t, "0", count)
}
