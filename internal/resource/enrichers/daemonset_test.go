package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestDaemonSetEnricher_NodeSelectorDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"nodeSelector": map[string]any{
				"zone":     "us-east",
				"disktype": "ssd",
			},
		},
		"status": map[string]any{
			"numberReady":            int64(2),
			"desiredNumberScheduled": int64(3),
		},
	}}

	e := &enrichers.DaemonSetEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "nodeSelectorDisplay")
	testza.AssertEqual(t, "disktype=ssd,zone=us-east", display)
}

func TestDaemonSetEnricher_EmptyNodeSelector(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"numberReady":            int64(0),
			"desiredNumberScheduled": int64(0),
		},
	}}

	e := &enrichers.DaemonSetEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "nodeSelectorDisplay")
	testza.AssertEqual(t, "", display)
}
