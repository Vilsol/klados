package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestServiceAccountEnricher_SecretsCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"secrets": []any{
			map[string]any{"name": "sa-token-abc"},
			map[string]any{"name": "sa-token-xyz"},
		},
	}}

	e := &enrichers.ServiceAccountEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "secretsCount")
	testza.AssertEqual(t, int64(2), count)
}

func TestServiceAccountEnricher_NoSecrets(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{}}

	e := &enrichers.ServiceAccountEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "secretsCount")
	testza.AssertEqual(t, int64(0), count)
}
