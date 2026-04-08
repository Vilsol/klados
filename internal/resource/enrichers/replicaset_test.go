package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestReplicaSetEnricher_OwnerDisplay(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"ownerReferences": []any{
				map[string]any{"kind": "Deployment", "name": "my-deployment"},
			},
		},
	}}

	e := &enrichers.ReplicaSetEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "ownerDisplay")
	testza.AssertEqual(t, "my-deployment", display)
}

func TestReplicaSetEnricher_NoOwner(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{},
	}}

	e := &enrichers.ReplicaSetEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	display, _, _ := unstructured.NestedString(obj.Object, "status", "ownerDisplay")
	testza.AssertEqual(t, "<none>", display)
}
