package resource_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/resource"
)

type trackingEnricher struct {
	key   string
	value string
	order *[]string
}

func (e *trackingEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	if e.order != nil {
		*e.order = append(*e.order, e.key)
	}
	obj.Object[e.key] = e.value
	return nil
}

func TestEnricherRegistry_Chaining_OrderPreserved(t *testing.T) {
	reg := resource.NewEnricherRegistry()

	var order []string
	reg.Register("apps.v1.deployments", &trackingEnricher{key: "first", value: "1", order: &order})
	reg.Register("apps.v1.deployments", &trackingEnricher{key: "second", value: "2", order: &order})

	obj := &unstructured.Unstructured{Object: map[string]any{"kind": "Deployment"}}
	for _, e := range reg.GetAll("apps.v1.deployments") {
		testza.AssertNil(t, e.Enrich("", obj))
	}

	testza.AssertEqual(t, "1", obj.Object["first"])
	testza.AssertEqual(t, "2", obj.Object["second"])
	testza.AssertEqual(t, []string{"first", "second"}, order)
}

type readingEnricher struct{}

func (e *readingEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	if obj.Object["base"] == "set" {
		obj.Object["derived"] = "yes"
	}
	return nil
}

func TestEnricherRegistry_SecondEnricherSeesFirstOutput(t *testing.T) {
	reg := resource.NewEnricherRegistry()
	reg.Register("core.v1.pods", &trackingEnricher{key: "base", value: "set"})
	reg.Register("core.v1.pods", &readingEnricher{})

	obj := &unstructured.Unstructured{Object: map[string]any{}}
	for _, e := range reg.GetAll("core.v1.pods") {
		testza.AssertNil(t, e.Enrich("", obj))
	}

	testza.AssertEqual(t, "set", obj.Object["base"])
	testza.AssertEqual(t, "yes", obj.Object["derived"])
}

func TestEnricherRegistry_DifferentGVRsAreIndependent(t *testing.T) {
	reg := resource.NewEnricherRegistry()
	reg.Register("apps.v1.deployments", &trackingEnricher{key: "fromDeploy", value: "yes"})
	reg.Register("core.v1.pods", &trackingEnricher{key: "fromPod", value: "yes"})

	obj := &unstructured.Unstructured{Object: map[string]any{}}
	for _, e := range reg.GetAll("apps.v1.deployments") {
		_ = e.Enrich("", obj)
	}
	testza.AssertEqual(t, "yes", obj.Object["fromDeploy"])
	testza.AssertNil(t, obj.Object["fromPod"])
}
