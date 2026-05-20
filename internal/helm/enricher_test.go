package helm

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestEnricher_PopulatesDisplayFields(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"chart":        "nginx",
			"chartVersion": "15.4.4",
			"appVersion":   "1.27.0",
			"revision":     int64(4),
			"status":       "pending-upgrade",
			"deployedAt":   "2026-05-19T10:00:00Z",
		},
	}}
	e := NewEnricher()
	err := e.Enrich("ctx1", u)
	testza.AssertNoError(t, err)
	status := u.Object["status"].(map[string]any)
	testza.AssertEqual(t, "Pending Upgrade", status["statusDisplay"])
	testza.AssertEqual(t, "rev 4", status["revisionDisplay"])
	testza.AssertEqual(t, "nginx-15.4.4", status["chartDisplay"])
	testza.AssertEqual(t, "1.27.0", status["appVersion"])
	testza.AssertEqual(t, "2026-05-19T10:00:00Z", status["lastDeployedDisplay"])
	testza.AssertEqual(t, int64(0), status["ownedResourceCount"])
}

func TestEnricher_SafeDefaults(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{}}
	e := NewEnricher()
	err := e.Enrich("ctx1", u)
	testza.AssertNoError(t, err)
	status := u.Object["status"].(map[string]any)
	testza.AssertEqual(t, "", status["statusDisplay"])
	testza.AssertEqual(t, "", status["revisionDisplay"])
	testza.AssertEqual(t, "", status["chartDisplay"])
}

func TestEnricher_NilInputs(t *testing.T) {
	e := NewEnricher()
	testza.AssertNoError(t, e.Enrich("ctx1", nil))
	testza.AssertNoError(t, e.Enrich("ctx1", &unstructured.Unstructured{}))
}
