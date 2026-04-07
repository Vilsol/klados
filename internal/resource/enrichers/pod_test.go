package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestPodEnricher(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "test"},
		"spec": map[string]any{
			"containers": []any{
				map[string]any{"name": "app"},
				map[string]any{"name": "sidecar"},
			},
		},
		"status": map[string]any{
			"phase": "Running",
			"containerStatuses": []any{
				map[string]any{"name": "app", "ready": true, "restartCount": int64(3)},
				map[string]any{"name": "sidecar", "ready": false, "restartCount": int64(1)},
			},
		},
	}}

	e := &enrichers.PodEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	readyDisplay, _, _ := unstructured.NestedString(obj.Object, "status", "readyDisplay")
	testza.AssertEqual(t, "1/2", readyDisplay)

	restartCount, _, _ := unstructured.NestedInt64(obj.Object, "status", "restartCount")
	testza.AssertEqual(t, int64(4), restartCount)

	statusDisplay, _, _ := unstructured.NestedString(obj.Object, "status", "statusDisplay")
	testza.AssertEqual(t, "Running", statusDisplay)
}

func TestPodEnricher_NoContainerStatuses(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"containers": []any{
				map[string]any{"name": "app"},
			},
		},
		"status": map[string]any{},
	}}

	e := &enrichers.PodEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	readyDisplay, _, _ := unstructured.NestedString(obj.Object, "status", "readyDisplay")
	testza.AssertEqual(t, "0/1", readyDisplay)
}
