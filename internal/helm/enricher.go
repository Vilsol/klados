package helm

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Enricher implements resource.Enricher: it injects display-only fields under
// status.* on virtual Helm release objects.
type Enricher struct{}

// NewEnricher constructs a Helm Enricher.
func NewEnricher() *Enricher {
	return &Enricher{}
}

// Enrich populates status.{statusDisplay,revisionDisplay,chartDisplay,appVersion,lastDeployedDisplay,ownedResourceCount}.
// The contextName argument is unused but required by the resource.Enricher
// signature.
func (e *Enricher) Enrich(_ string, u *unstructured.Unstructured) error {
	if u == nil || u.Object == nil {
		return nil
	}
	spec, _, _ := unstructured.NestedMap(u.Object, "spec")
	status, _, _ := unstructured.NestedMap(u.Object, "status")
	if status == nil {
		status = map[string]any{}
	}

	statusStr, _ := spec["status"].(string)
	status["statusDisplay"] = humanStatus(statusStr)

	switch v := spec["revision"].(type) {
	case int64:
		status["revisionDisplay"] = fmt.Sprintf("rev %d", v)
	case int:
		status["revisionDisplay"] = fmt.Sprintf("rev %d", v)
	case float64:
		status["revisionDisplay"] = fmt.Sprintf("rev %d", int(v))
	default:
		status["revisionDisplay"] = ""
	}

	chartName, _ := spec["chart"].(string)
	chartVer, _ := spec["chartVersion"].(string)
	if chartName != "" && chartVer != "" {
		status["chartDisplay"] = chartName + "-" + chartVer
	} else if chartName != "" {
		status["chartDisplay"] = chartName
	} else {
		status["chartDisplay"] = ""
	}

	appVer, _ := spec["appVersion"].(string)
	status["appVersion"] = appVer

	deployedAt, _ := spec["deployedAt"].(string)
	status["lastDeployedDisplay"] = deployedAt

	if _, ok := status["ownedResourceCount"]; !ok {
		status["ownedResourceCount"] = int64(0)
	}

	u.Object["status"] = status
	return nil
}

// humanStatus formats a Helm release status into a UI-friendly string.
func humanStatus(s string) string {
	if s == "" {
		return ""
	}
	// Helm uses lowercase "deployed" / "pending-upgrade" etc.
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
