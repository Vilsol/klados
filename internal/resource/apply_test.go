package resource_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/Vilsol/klados/internal/resource"
)

func makeUnstructuredPod(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
	}}
}

func newApplyEngine() (*resource.ResourceEngine, *fake.FakeDynamicClient) {
	sc := runtime.NewScheme()
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(sc, map[schema.GroupVersionResource]string{
		podGVR: "PodList",
	})
	enricherReg := resource.NewEnricherRegistry()
	return resource.NewResourceEngine(&fakeProvider{dyn}, enricherReg), dyn
}

func TestResourceEngine_Apply_UsesApplyPatchType(t *testing.T) {
	engine, dyn := newApplyEngine()

	var capturedPatchType types.PatchType
	var capturedFieldManager string
	var capturedForce *bool
	dyn.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		pa := action.(k8stesting.PatchAction)
		capturedPatchType = pa.GetPatchType()
		if impl, ok := action.(k8stesting.PatchActionImpl); ok {
			capturedFieldManager = impl.PatchOptions.FieldManager
			capturedForce = impl.PatchOptions.Force
		}
		return true, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]any{"name": pa.GetName(), "namespace": pa.GetNamespace()},
		}}, nil
	})
	dyn.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, action.(k8stesting.GetAction).GetName())
	})

	obj := makeUnstructuredPod("test-pod", "default")
	result, err := engine.Apply(context.Background(), "ctx", "core.v1.pods", obj)

	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, result)
	testza.AssertEqual(t, types.ApplyPatchType, capturedPatchType)
	testza.AssertEqual(t, "klados", capturedFieldManager)
	testza.AssertNotNil(t, capturedForce)
	testza.AssertTrue(t, *capturedForce)
}

func TestResourceEngine_Apply_ActionCreated_WhenResourceNotFound(t *testing.T) {
	engine, dyn := newApplyEngine()

	dyn.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, action.(k8stesting.GetAction).GetName())
	})
	dyn.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		pa := action.(k8stesting.PatchAction)
		var raw map[string]any
		_ = json.Unmarshal(pa.GetPatch(), &raw)
		return true, &unstructured.Unstructured{Object: raw}, nil
	})

	result, err := engine.Apply(context.Background(), "ctx", "core.v1.pods", makeUnstructuredPod("new-pod", "default"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "created", result.Action)
	testza.AssertEqual(t, "", result.Error)
}

func TestResourceEngine_Apply_ActionConfigured_WhenResourceExists(t *testing.T) {
	engine, dyn := newApplyEngine()

	dyn.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]any{"name": action.(k8stesting.GetAction).GetName()},
		}}, nil
	})
	dyn.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		pa := action.(k8stesting.PatchAction)
		var raw map[string]any
		_ = json.Unmarshal(pa.GetPatch(), &raw)
		return true, &unstructured.Unstructured{Object: raw}, nil
	})

	result, err := engine.Apply(context.Background(), "ctx", "core.v1.pods", makeUnstructuredPod("existing-pod", "default"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "configured", result.Action)
}

func TestResourceEngine_Apply_PatchError_NonFatal(t *testing.T) {
	engine, dyn := newApplyEngine()

	dyn.PrependReactor("get", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "bad-pod")
	})
	dyn.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewBadRequest("invalid resource")
	})

	result, err := engine.Apply(context.Background(), "ctx", "core.v1.pods", makeUnstructuredPod("bad-pod", "default"))
	testza.AssertNoError(t, err) // Go error is nil; failure is in result
	testza.AssertNotNil(t, result)
	testza.AssertNotEqual(t, "", result.Error)
}
