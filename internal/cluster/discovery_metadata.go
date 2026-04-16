package cluster

import (
	"strings"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DetectSubresources inspects an APIResourceList and returns a map keyed by
// parent resource name. Each entry records whether well-known subresources
// (scale, status) are served. Kubernetes exposes subresources as separate
// entries like "deployments/scale"; we group them back under the parent.
func DetectSubresources(list *metav1.APIResourceList) map[string]ResourceSubresources {
	out := map[string]ResourceSubresources{}
	if list == nil {
		return out
	}
	for _, r := range list.APIResources {
		name := r.Name
		if idx := strings.Index(name, "/"); idx >= 0 {
			parent := name[:idx]
			sub := name[idx+1:]
			entry := out[parent]
			switch sub {
			case "scale":
				entry.Scale = true
			case "status":
				entry.Status = true
			}
			out[parent] = entry
		} else if _, ok := out[name]; !ok {
			// Ensure parent is present even if it has no subresources yet.
			out[name] = ResourceSubresources{}
		}
	}
	return out
}

// CRDMetadata is the per-GVR data extracted from a CRD object.
type CRDMetadata struct {
	PrinterColumns []AdditionalPrinterColumn
	ScaleSpec      *ScaleSubresourceSpec
}

// ExtractCRDMetadata walks a list of CRDs and produces a per-GVR metadata
// map keyed by GVR in the same dot-separated format as APIResource.GVR.
// Only served versions are included.
func ExtractCRDMetadata(crds []apiextv1.CustomResourceDefinition) map[string]CRDMetadata {
	out := map[string]CRDMetadata{}
	for _, crd := range crds {
		group := crd.Spec.Group
		plural := crd.Spec.Names.Plural
		for _, v := range crd.Spec.Versions {
			if !v.Served {
				continue
			}
			gvr := formatGVR(group, v.Name, plural)

			md := CRDMetadata{}
			for _, c := range v.AdditionalPrinterColumns {
				md.PrinterColumns = append(md.PrinterColumns, AdditionalPrinterColumn{
					Name:        c.Name,
					Type:        c.Type,
					Format:      c.Format,
					Description: c.Description,
					Priority:    c.Priority,
					JSONPath:    c.JSONPath,
				})
			}
			if v.Subresources != nil && v.Subresources.Scale != nil {
				spec := v.Subresources.Scale.SpecReplicasPath
				status := v.Subresources.Scale.StatusReplicasPath
				if spec == "" {
					spec = ".spec.replicas"
				}
				if status == "" {
					status = ".status.replicas"
				}
				md.ScaleSpec = &ScaleSubresourceSpec{
					SpecReplicasPath:   spec,
					StatusReplicasPath: status,
				}
			}
			out[gvr] = md
		}
	}
	return out
}

// formatGVR produces the dot-separated GVR string used elsewhere in the
// codebase (e.g. "example.com.v1.widgets", "core.v1.pods"). An empty group
// becomes "core" to match built-in convention.
func formatGVR(group, version, resource string) string {
	if group == "" {
		group = "core"
	}
	return group + "." + version + "." + resource
}
