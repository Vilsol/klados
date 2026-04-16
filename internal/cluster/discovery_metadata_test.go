package cluster

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetectSubresources_FromAPIResourceList(t *testing.T) {
	list := &metav1.APIResourceList{
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Namespaced: true, Kind: "Deployment"},
			{Name: "deployments/scale", Namespaced: true, Kind: "Scale"},
			{Name: "deployments/status", Namespaced: true, Kind: "Deployment"},
			{Name: "replicasets", Namespaced: true, Kind: "ReplicaSet"},
			{Name: "replicasets/scale", Namespaced: true, Kind: "Scale"},
			{Name: "statefulsets", Namespaced: true, Kind: "StatefulSet"},
		},
	}

	subs := DetectSubresources(list)

	testza.AssertTrue(t, subs["deployments"].Scale)
	testza.AssertTrue(t, subs["deployments"].Status)
	testza.AssertTrue(t, subs["replicasets"].Scale)
	testza.AssertFalse(t, subs["replicasets"].Status)
	testza.AssertFalse(t, subs["statefulsets"].Scale)
}

func TestDetectSubresources_Empty(t *testing.T) {
	subs := DetectSubresources(&metav1.APIResourceList{})
	testza.AssertEqual(t, 0, len(subs))
}

func TestExtractCRDMetadata_PrinterColumnsAndScale(t *testing.T) {
	served := true
	crd := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "widgets", Kind: "Widget"},
			Scope: apiextv1.NamespaceScoped,
			Versions: []apiextv1.CustomResourceDefinitionVersion{{
				Name:   "v1",
				Served: served,
				AdditionalPrinterColumns: []apiextv1.CustomResourceColumnDefinition{
					{Name: "Replicas", Type: "integer", JSONPath: ".spec.replicas"},
					{Name: "Ready", Type: "string", JSONPath: ".status.ready", Priority: 1},
				},
				Subresources: &apiextv1.CustomResourceSubresources{
					Scale: &apiextv1.CustomResourceSubresourceScale{
						SpecReplicasPath:   ".spec.size",
						StatusReplicasPath: ".status.currentSize",
					},
					Status: &apiextv1.CustomResourceSubresourceStatus{},
				},
			}},
		},
	}

	md := ExtractCRDMetadata([]apiextv1.CustomResourceDefinition{crd})

	gvr := "example.com.v1.widgets"
	entry, ok := md[gvr]
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, 2, len(entry.PrinterColumns))
	testza.AssertEqual(t, "Replicas", entry.PrinterColumns[0].Name)
	testza.AssertEqual(t, ".spec.replicas", entry.PrinterColumns[0].JSONPath)
	testza.AssertEqual(t, int32(1), entry.PrinterColumns[1].Priority)
	testza.AssertNotNil(t, entry.ScaleSpec)
	testza.AssertEqual(t, ".spec.size", entry.ScaleSpec.SpecReplicasPath)
	testza.AssertEqual(t, ".status.currentSize", entry.ScaleSpec.StatusReplicasPath)
}

func TestExtractCRDMetadata_DefaultScalePaths(t *testing.T) {
	crd := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "things"},
			Versions: []apiextv1.CustomResourceDefinitionVersion{{
				Name: "v1", Served: true,
				Subresources: &apiextv1.CustomResourceSubresources{
					Scale: &apiextv1.CustomResourceSubresourceScale{},
				},
			}},
		},
	}

	md := ExtractCRDMetadata([]apiextv1.CustomResourceDefinition{crd})
	entry := md["example.com.v1.things"]
	testza.AssertEqual(t, ".spec.replicas", entry.ScaleSpec.SpecReplicasPath)
	testza.AssertEqual(t, ".status.replicas", entry.ScaleSpec.StatusReplicasPath)
}

func TestExtractCRDMetadata_SkipsUnservedVersions(t *testing.T) {
	crd := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextv1.CustomResourceDefinitionNames{Plural: "widgets"},
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{Name: "v1alpha1", Served: false},
				{Name: "v1", Served: true},
			},
		},
	}

	md := ExtractCRDMetadata([]apiextv1.CustomResourceDefinition{crd})
	_, hasAlpha := md["example.com.v1alpha1.widgets"]
	_, hasV1 := md["example.com.v1.widgets"]
	testza.AssertFalse(t, hasAlpha)
	testza.AssertTrue(t, hasV1)
}

func TestFormatGVR_EmptyGroupBecomesCore(t *testing.T) {
	testza.AssertEqual(t, "core.v1.pods", formatGVR("", "v1", "pods"))
	testza.AssertEqual(t, "apps.v1.deployments", formatGVR("apps", "v1", "deployments"))
}
