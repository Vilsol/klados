package enrichers_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource/enrichers"
)

func TestLeaseEnricher_LeaseDurationDisplay_Seconds(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"leaseDurationSeconds": int64(30),
		},
	}}
	e := &enrichers.LeaseEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "leaseDurationDisplay")
	testza.AssertEqual(t, "30s", display)
}

func TestLeaseEnricher_LeaseDurationDisplay_Minutes(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"leaseDurationSeconds": int64(120),
		},
	}}
	e := &enrichers.LeaseEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "leaseDurationDisplay")
	testza.AssertEqual(t, "2m", display)
}

func TestLeaseEnricher_LeaseDurationDisplay_Nil(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{},
	}}
	e := &enrichers.LeaseEnricher{}
	testza.AssertNoError(t, e.Enrich("", obj))
	display, _, _ := unstructured.NestedString(obj.Object, "status", "leaseDurationDisplay")
	testza.AssertEqual(t, "", display)
}
