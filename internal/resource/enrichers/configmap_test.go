package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestConfigMapEnricher_DataKeysCount(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"data": map[string]any{
			"key1": "val1",
			"key2": "val2",
			"key3": "val3",
		},
	}}

	e := &enrichers.ConfigMapEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "dataKeysCount")
	testza.AssertEqual(t, int64(3), count)
}

func TestConfigMapEnricher_EmptyData(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{}}

	e := &enrichers.ConfigMapEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))

	count, _, _ := unstructured.NestedInt64(obj.Object, "status", "dataKeysCount")
	testza.AssertEqual(t, int64(0), count)
}
