package resource_test

import (
	"encoding/json"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/klados/internal/resource"
)

func TestDescriptor_GVR_CoreGroup(t *testing.T) {
	d := &resource.Descriptor{Group: "", Version: "v1", Resource: "pods"}
	testza.AssertEqual(t, "core.v1.pods", d.GVR())
}

func TestDescriptor_GVR_NamedGroup(t *testing.T) {
	d := &resource.Descriptor{Group: "apps", Version: "v1", Resource: "deployments"}
	testza.AssertEqual(t, "apps.v1.deployments", d.GVR())
}

func TestDescriptor_GVR_DottedGroup(t *testing.T) {
	d := &resource.Descriptor{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
	testza.AssertEqual(t, "networking.k8s.io.v1.ingresses", d.GVR())
}

func TestNewRegistry(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, reg)
}

func TestRegistry_Register_And_Get(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	d := &resource.Descriptor{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
		Columns: []resource.Column{
			{Name: "Name", Expr: `metadata.name`, RenderType: resource.RenderText},
		},
	}

	testza.AssertNoError(t, reg.Register(d))

	got, ok := reg.Get("apps.v1.deployments")
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, d, got)
}

func TestRegistry_Get_Missing(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	_, ok := reg.Get("core.v1.pods")
	testza.AssertFalse(t, ok)
}

func TestRegistry_List(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	testza.AssertEqual(t, 0, len(reg.List()))

	d1 := &resource.Descriptor{Group: "", Version: "v1", Resource: "pods"}
	d2 := &resource.Descriptor{Group: "apps", Version: "v1", Resource: "deployments"}
	testza.AssertNoError(t, reg.Register(d1))
	testza.AssertNoError(t, reg.Register(d2))

	testza.AssertEqual(t, 2, len(reg.List()))
}

func TestRegistry_Register_CEL_Error(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	d := &resource.Descriptor{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
		Columns: []resource.Column{
			{Name: "Bad", Expr: `!!!invalid cel!!!`, RenderType: resource.RenderText},
		},
	}

	err = reg.Register(d)
	testza.AssertNotNil(t, err)
}

func TestColumnAlignDefault(t *testing.T) {
	col := resource.Column{Name: "Name", Expr: "metadata.name", RenderType: resource.RenderText}
	data, err := json.Marshal(col)
	testza.AssertNoError(t, err)
	var m map[string]any
	testza.AssertNoError(t, json.Unmarshal(data, &m))
	_, hasAlign := m["align"]
	testza.AssertFalse(t, hasAlign)
}

func TestNamespaceColumnOnAllNamespacedDescriptors(t *testing.T) {
	for _, d := range resource.BuiltinDescriptors() {
		if d.ClusterScoped {
			continue
		}
		found := false
		for _, col := range d.Columns {
			if col.Name == "Namespace" {
				testza.AssertTrue(t, col.Hidden, "Namespace column on %s must be Hidden", d.GVR())
				found = true
				break
			}
		}
		testza.AssertTrue(t, found, "descriptor %s missing Namespace column", d.GVR())
	}
}

func TestClusterScopedDescriptorsHaveNoNamespaceColumn(t *testing.T) {
	for _, d := range resource.BuiltinDescriptors() {
		if !d.ClusterScoped {
			continue
		}
		for _, col := range d.Columns {
			testza.AssertNotEqual(t, "Namespace", col.Name, "cluster-scoped descriptor %s must not have Namespace column", d.GVR())
		}
	}
}

func TestRegistry_Register_OverviewField_CEL_Error(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	d := &resource.Descriptor{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
		OverviewFields: []resource.OverviewField{
			{Label: "Bad", Expr: `!!!invalid!!!`, RenderType: resource.RenderText},
		},
	}

	err = reg.Register(d)
	testza.AssertNotNil(t, err)
}
