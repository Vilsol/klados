package resource_test

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/resource"
	k8sfake "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestTemplateRegistry_RegisterAndGet(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	tmpl := resource.Template{GVR: "core.v1.pods", Name: "Basic Pod", Source: "builtin", Content: "apiVersion: v1\nkind: Pod\n"}
	reg.Register(tmpl)

	got := reg.GetTemplates("core.v1.pods")
	testza.AssertLen(t, got, 1)
	testza.AssertEqual(t, "Basic Pod", got[0].Name)
	testza.AssertEqual(t, "builtin", got[0].Source)
}

func TestTemplateRegistry_GetReturnsEmpty(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	got := reg.GetTemplates("apps.v1.deployments")
	testza.AssertLen(t, got, 0)
}

func TestTemplateRegistry_Plugin_RegisterAndGet(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	tmpl := resource.Template{GVR: "apps.v1.deployments", Name: "Plugin Deploy", Source: "plugin:istio"}
	reg.RegisterPlugin("istio", tmpl)

	got := reg.GetTemplates("apps.v1.deployments")
	testza.AssertLen(t, got, 1)
	testza.AssertEqual(t, "plugin:istio", got[0].Source)
}

func TestTemplateRegistry_Plugin_UnregisterRemovesOnly(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	reg.Register(resource.Template{GVR: "apps.v1.deployments", Name: "Builtin", Source: "builtin"})
	reg.RegisterPlugin("istio", resource.Template{GVR: "apps.v1.deployments", Name: "Istio", Source: "plugin:istio"})
	reg.RegisterPlugin("other", resource.Template{GVR: "apps.v1.deployments", Name: "Other", Source: "plugin:other"})

	reg.UnregisterPlugin("istio")

	got := reg.GetTemplates("apps.v1.deployments")
	testza.AssertLen(t, got, 2)
	for _, t2 := range got {
		testza.AssertNotEqual(t, "Istio", t2.Name)
	}
}

func TestBuiltinTemplates_AllParseable(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	err := resource.LoadBuiltinTemplates(reg)
	testza.AssertNoError(t, err)

	gvrs := reg.GetAllGVRs()
	testza.AssertTrue(t, len(gvrs) >= 20, "expected at least 20 GVRs from builtin templates")
}

func TestGenerateFromSchema_ReturnsValidSkeleton(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	disc := &k8sfake.FakeDiscovery{Fake: &k8stesting.Fake{}}
	tmpl, err := reg.GenerateFromSchema(context.Background(), "apps.v1.deployments", disc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "schema", tmpl.Source)
	testza.AssertEqual(t, "apps.v1.deployments", tmpl.GVR)
	testza.AssertContains(t, tmpl.Content, "apiVersion:")
	testza.AssertContains(t, tmpl.Content, "kind:")
	testza.AssertContains(t, tmpl.Content, "metadata:")
}

func TestGenerateFromSchema_FallbackForUnknownCRD(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	disc := &k8sfake.FakeDiscovery{Fake: &k8stesting.Fake{}}
	tmpl, err := reg.GenerateFromSchema(context.Background(), "example.com.v1.widgets", disc)
	testza.AssertNoError(t, err)
	testza.AssertContains(t, tmpl.Content, "apiVersion:")
	testza.AssertContains(t, tmpl.Content, "kind:")
	testza.AssertContains(t, tmpl.Content, "metadata:")
}

func TestTemplateRegistry_GetAllGVRs(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	reg.Register(resource.Template{GVR: "core.v1.pods", Name: "Pod", Source: "builtin"})
	reg.Register(resource.Template{GVR: "apps.v1.deployments", Name: "Deploy", Source: "builtin"})
	reg.RegisterPlugin("istio", resource.Template{GVR: "networking.istio.io.v1.virtualservices", Name: "VS", Source: "plugin:istio"})

	gvrs := reg.GetAllGVRs()
	testza.AssertContains(t, gvrs, "core.v1.pods")
	testza.AssertContains(t, gvrs, "apps.v1.deployments")
	testza.AssertContains(t, gvrs, "networking.istio.io.v1.virtualservices")
}
