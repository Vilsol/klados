package enrichers

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var accessModeAbbrev = map[string]string{
	"ReadWriteOnce":    "RWO",
	"ReadOnlyMany":     "ROX",
	"ReadWriteMany":    "RWX",
	"ReadWriteOncePod": "RWOP",
}

func abbreviateAccessModes(modes []string) string {
	parts := make([]string, 0, len(modes))
	for _, m := range modes {
		if abbr, ok := accessModeAbbrev[m]; ok {
			parts = append(parts, abbr)
		} else {
			parts = append(parts, m)
		}
	}
	return strings.Join(parts, ",")
}

type PVEnricher struct{}

func (e *PVEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
	modes, _, _ := unstructured.NestedStringSlice(obj.Object, "spec", "accessModes")
	_ = unstructured.SetNestedField(obj.Object, abbreviateAccessModes(modes), "status", "accessModesDisplay")

	ns, _, _ := unstructured.NestedString(obj.Object, "spec", "claimRef", "namespace")
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "claimRef", "name")
	claimDisplay := ""
	if ns != "" || name != "" {
		claimDisplay = ns + "/" + name
	}
	_ = unstructured.SetNestedField(obj.Object, claimDisplay, "status", "claimDisplay")

	return nil
}
