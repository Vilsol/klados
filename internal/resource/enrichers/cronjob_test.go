package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestCronJobEnricher_ActiveCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"active": []any{
				map[string]any{"name": "job-1"},
				map[string]any{"name": "job-2"},
			},
		},
	}}

	e := &enrichers.CronJobEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "activeCount")
	testza.AssertEqual(t, int64(2), count)
}

func TestCronJobEnricher_NoActive(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{},
	}}

	e := &enrichers.CronJobEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "activeCount")
	testza.AssertEqual(t, int64(0), count)
}
