package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestSecretEnricher_DataKeysCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"data": map[string]any{
			"username": "dXNlcg==",
			"password": "cGFzcw==",
		},
	}}

	e := &enrichers.SecretEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "dataKeysCount")
	testza.AssertEqual(t, int64(2), count)
}
