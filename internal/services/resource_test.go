package services

import (
	"context"
	"testing"

	"github.com/MarvinJWendt/testza"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/resource"
)

type fakeConnProvider struct {
	dynamic *fake.FakeDynamicClient
}

func (f *fakeConnProvider) GetConnection(_ string) (*cluster.Connection, error) {
	conn := &cluster.Connection{}
	conn.Dynamic = f.dynamic
	return conn, nil
}

func newTestResourceService(objects ...runtime.Object) *ResourceService {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	deplGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		deplGVR: "DeploymentList",
	}, objects...)

	enricherReg := resource.NewEnricherRegistry()
	eng := resource.NewResourceEngine(&fakeConnProvider{dyn}, enricherReg)

	return &ResourceService{
		engine: eng,
		ctx:    context.Background(),
	}
}

func makeDeployment(name, namespace string, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}

func TestResourceService_ScaleResource(t *testing.T) {
	svc := newTestResourceService(makeDeployment("my-deploy", "default", 2))

	err := svc.ScaleResource("ctx", "apps.v1.deployments", "default", "my-deploy", 5)
	testza.AssertNoError(t, err)
}

func TestResourceService_RestartResource(t *testing.T) {
	svc := newTestResourceService(makeDeployment("my-deploy", "default", 1))

	err := svc.RestartResource("ctx", "apps.v1.deployments", "default", "my-deploy")
	testza.AssertNoError(t, err)
}

func TestResourceService_UpdateResource(t *testing.T) {
	deplGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	depl := makeDeployment("upd-deploy", "default", 1)
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		deplGVR: "DeploymentList",
	}, depl)

	enricherReg := resource.NewEnricherRegistry()
	eng := resource.NewResourceEngine(&fakeConnProvider{dyn}, enricherReg)

	svc := &ResourceService{engine: eng, ctx: context.Background()}

	obj, err := svc.GetResource("ctx", "apps.v1.deployments", "default", "upd-deploy")
	testza.AssertNoError(t, err)

	meta := obj["metadata"].(map[string]any)
	if meta["labels"] == nil {
		meta["labels"] = map[string]any{}
	}
	meta["labels"].(map[string]any)["test"] = "true"

	updated, err := svc.UpdateResource("ctx", "apps.v1.deployments", "default", obj)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, updated)
}

func TestResourceService_ForceDeleteResource(t *testing.T) {
	deplGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	depl := makeDeployment("del-deploy", "default", 1)
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		deplGVR: "DeploymentList",
	}, depl)

	enricherReg := resource.NewEnricherRegistry()
	eng := resource.NewResourceEngine(&fakeConnProvider{dyn}, enricherReg)

	svc := &ResourceService{engine: eng, ctx: context.Background()}

	err := svc.ForceDeleteResource("ctx", "apps.v1.deployments", "default", "del-deploy")
	testza.AssertNoError(t, err)

	list, err := dyn.Resource(deplGVR).Namespace("default").List(context.Background(), metav1.ListOptions{})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 0, len(list.Items))
}

func newTestResourceServiceCRD() *ResourceService {
	scheme := runtime.NewScheme()
	crdGVR := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		crdGVR: "WidgetList",
	})
	enricherReg := resource.NewEnricherRegistry()
	eng := resource.NewResourceEngine(&fakeConnProvider{dyn}, enricherReg)
	return &ResourceService{engine: eng, ctx: context.Background()}
}

func TestResourceService_CreateResource(t *testing.T) {
	svc := newTestResourceServiceCRD()

	obj := map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata":   map[string]any{"name": "test-widget", "namespace": "default"},
	}

	result, err := svc.CreateResource("ctx", "example.com.v1.widgets", "default", obj)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, result)
	meta := result["metadata"].(map[string]any)
	testza.AssertEqual(t, "test-widget", meta["name"])
}

func TestResourceService_ListResources(t *testing.T) {
	svc := newTestResourceService(makeDeployment("my-deploy", "default", 1))

	items, err := svc.ListResources("ctx", "apps.v1.deployments", "default")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 1, len(items))
}

func TestResourceService_GetResource(t *testing.T) {
	svc := newTestResourceService(makeDeployment("my-deploy", "default", 1))

	obj, err := svc.GetResource("ctx", "apps.v1.deployments", "default", "my-deploy")
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, obj)
}

func TestResourceService_DeleteResource(t *testing.T) {
	deplGVR := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	svc := newTestResourceService(makeDeployment("del-deploy", "default", 1))

	err := svc.DeleteResource("ctx", "apps.v1.deployments", "default", "del-deploy")
	testza.AssertNoError(t, err)

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		deplGVR: "DeploymentList",
	})
	eng := resource.NewResourceEngine(&fakeConnProvider{dyn}, resource.NewEnricherRegistry())
	svc2 := &ResourceService{engine: eng, ctx: context.Background()}
	list, err := svc2.ListResources("ctx", "apps.v1.deployments", "default")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 0, len(list))
}
