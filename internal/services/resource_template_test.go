package services_test

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/resource"
	"github.com/Vilsol/klados/internal/services"
)

func TestGetTemplates_ReturnsCuratedTemplates(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	reg.Register(resource.Template{
		GVR:     "core.v1.pods",
		Name:    "Basic Pod",
		Source:  "builtin",
		Content: "apiVersion: v1\nkind: Pod\n",
	})

	svc := services.NewResourceServiceWithRegistry(reg)
	templates, err := svc.GetTemplates("test-ctx", "core.v1.pods")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, templates, 1)
	testza.AssertEqual(t, "Basic Pod", templates[0].Name)
}

func TestGetTemplates_FallsBackToSkeletonWhenNoAppService(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	svc := services.NewResourceServiceWithRegistry(reg)
	templates, err := svc.GetTemplates("test-ctx", "apps.v1.deployments")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, templates, 1)
	testza.AssertEqual(t, "Default", templates[0].Name)
	testza.AssertEqual(t, "schema", templates[0].Source)
}

func TestGetAllTemplateGVRs_IncludesBuiltins(t *testing.T) {
	reg := resource.NewTemplateRegistry()
	reg.Register(resource.Template{GVR: "core.v1.pods", Name: "Pod", Source: "builtin"})
	reg.Register(resource.Template{GVR: "apps.v1.deployments", Name: "Deploy", Source: "builtin"})

	svc := services.NewResourceServiceWithRegistry(reg)
	gvrs, err := svc.GetAllTemplateGVRs("test-ctx")
	testza.AssertNoError(t, err)
	testza.AssertContains(t, gvrs, "core.v1.pods")
	testza.AssertContains(t, gvrs, "apps.v1.deployments")
}
