package helm

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	"helm.sh/helm/v4/pkg/release/common"
	corev1 "k8s.io/api/core/v1"
)

const fixtureManifest = `# heading
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  annotations:
    helm.sh/resource-policy: keep
spec:
  replicas: 2
---

---
# comment-only doc
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  namespace: other-ns
spec:
  ports: []
`

func TestParseManifest_ExtractsResources(t *testing.T) {
	refs := parseManifest(fixtureManifest, "default")
	testza.AssertEqual(t, 2, len(refs))

	dep := refs[0]
	if dep.Kind != "Deployment" {
		dep = refs[1]
	}
	testza.AssertEqual(t, "Deployment", dep.Kind)
	testza.AssertEqual(t, "apps.v1.deployments", dep.GVR)
	testza.AssertEqual(t, "keep", dep.ResourcePolicy)
	testza.AssertEqual(t, "default", dep.Namespace) // default fallback

	svc := refs[0]
	if svc.Kind != "Service" {
		svc = refs[1]
	}
	testza.AssertEqual(t, "Service", svc.Kind)
	testza.AssertEqual(t, "core.v1.services", svc.GVR)
	testza.AssertEqual(t, "other-ns", svc.Namespace) // explicit ns surfaces
}

func TestGetOwnedResources_LabelFallback(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "myrel", "default", 1, common.StatusDeployed, nil, fixtureManifest),
	})
	getter := newFakeResourceGetter()
	// Deployment from manifest exists.
	getter.exists["apps.v1.deployments|default|nginx"] = true
	// Service from manifest does NOT exist (will surface Exists=false).
	// Label-fallback turns up a ConfigMap that's NOT in the manifest.
	getter.byLabel["core.v1.configmaps|default"] = []map[string]any{
		{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "nginx-extra",
				"namespace": "default",
			},
		},
	}
	b := NewBackend(NewClientCache(), lister)
	refs, err := b.GetOwnedResources(context.Background(), "ctx1", "default", "myrel", false, getter)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 3, len(refs))

	var foundExtra, foundDep, foundSvc bool
	for _, r := range refs {
		switch r.Name {
		case "nginx-extra":
			foundExtra = true
			testza.AssertTrue(t, r.Exists)
		case "nginx":
			if r.Kind == "Deployment" {
				foundDep = true
				testza.AssertTrue(t, r.Exists)
				testza.AssertEqual(t, "keep", r.ResourcePolicy)
			}
			if r.Kind == "Service" {
				foundSvc = true
				testza.AssertEqual(t, "other-ns", r.Namespace)
				testza.AssertFalse(t, r.Exists)
			}
		}
	}
	testza.AssertTrue(t, foundExtra)
	testza.AssertTrue(t, foundDep)
	testza.AssertTrue(t, foundSvc)
}

func TestGetOwnedResources_ScanAll(t *testing.T) {
	lister := newFakeSecretLister()
	lister.put("ctx1", "default", []corev1.Secret{
		makeReleaseSecret(t, "myrel", "default", 1, common.StatusDeployed, nil, ""),
	})
	getter := newFakeResourceGetter()
	getter.knownGVRs = []string{
		"custom.example.com.v1.widgets",
	}
	getter.byLabel["custom.example.com.v1.widgets|default"] = []map[string]any{
		{
			"apiVersion": "custom.example.com/v1",
			"kind":       "Widget",
			"metadata":   map[string]any{"name": "w1", "namespace": "default"},
		},
	}
	b := NewBackend(NewClientCache(), lister)
	refs, err := b.GetOwnedResources(context.Background(), "ctx1", "default", "myrel", true, getter)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(refs))
	testza.AssertEqual(t, "Widget", refs[0].Kind)
}
