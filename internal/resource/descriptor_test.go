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
		for i, col := range d.Columns {
			if col.Name == "Namespace" {
				testza.AssertFalse(t, col.Hidden, "Namespace column on %s must not be Hidden", d.GVR())
				testza.AssertEqual(t, 1, i, "Namespace column on %s must be second (index 1)", d.GVR())
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

func TestDescriptor_VirtualFields_RoundTrip(t *testing.T) {
	d := &resource.Descriptor{
		Group: "helm", Version: "v1", Resource: "releases",
		IsVirtual:  true,
		GroupLabel: "Helm",
		Available:  true,
		Columns:    []resource.Column{{Name: "Name", Expr: "metadata.name", RenderType: resource.RenderText}},
	}
	data, err := json.Marshal(d)
	testza.AssertNoError(t, err)

	var out resource.Descriptor
	testza.AssertNoError(t, json.Unmarshal(data, &out))
	testza.AssertTrue(t, out.IsVirtual)
	testza.AssertEqual(t, "Helm", out.GroupLabel)
	testza.AssertTrue(t, out.Available)
}

func TestRegistry_Register_SetsAvailableByDefault(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	d := &resource.Descriptor{Group: "", Version: "v1", Resource: "pods"}
	testza.AssertNoError(t, reg.Register(d))

	got, ok := reg.Get("core.v1.pods")
	testza.AssertTrue(t, ok)
	testza.AssertTrue(t, got.Available)
}

func TestRegistry_SetAvailable(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)

	d := &resource.Descriptor{Group: "helm", Version: "v1", Resource: "releases", IsVirtual: true}
	testza.AssertNoError(t, reg.Register(d))

	reg.SetAvailable("helm.v1.releases", false)
	got, _ := reg.Get("helm.v1.releases")
	testza.AssertFalse(t, got.Available)

	reg.SetAvailable("helm.v1.releases", true)
	got, _ = reg.Get("helm.v1.releases")
	testza.AssertTrue(t, got.Available)

	// no-op on unknown GVR
	reg.SetAvailable("does.not.exist", false)
}

func TestRegistry_SetAvailable_Concurrent(t *testing.T) {
	reg, err := resource.NewRegistry()
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, reg.Register(&resource.Descriptor{Group: "helm", Version: "v1", Resource: "releases"}))

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			reg.SetAvailable("helm.v1.releases", i%2 == 0)
		}
		close(done)
	}()
	for i := 0; i < 1000; i++ {
		_, _ = reg.Get("helm.v1.releases")
	}
	<-done
}

func TestBuiltin_HelmReleases_Registered(t *testing.T) {
	var found *resource.Descriptor
	for _, d := range resource.BuiltinDescriptors() {
		if d.GVR() == "helm.v1.releases" {
			found = d
			break
		}
	}
	testza.AssertNotNil(t, found)
	testza.AssertTrue(t, found.IsVirtual)
	testza.AssertEqual(t, "Helm", found.GroupLabel)
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
