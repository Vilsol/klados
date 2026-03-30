package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestDeploymentEnricher(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec":   map[string]any{"replicas": int64(3)},
		"status": map[string]any{"readyReplicas": int64(2), "availableReplicas": int64(2)},
	}}

	e := &enrichers.DeploymentEnricher{}
	testza.AssertNoError(t, e.Enrich(obj))

	readyDisplay, _, _ := unstructured.NestedString(obj.Object, "status", "readyDisplay")
	testza.AssertEqual(t, "2/3", readyDisplay)

	available, _, _ := unstructured.NestedInt64(obj.Object, "status", "availableReplicas")
	testza.AssertEqual(t, int64(2), available)
}
