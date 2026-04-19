package volumebrowser

import (
	"context"
	"fmt"
	"testing"

	"github.com/MarvinJWendt/testza"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/Vilsol/klados/internal/cluster"
)

func vbTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	// Register list kinds for the fake dynamic client.
	s.AddKnownTypeWithName(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PersistentVolumeClaimList"}, &unstructured.UnstructuredList{})
	s.AddKnownTypeWithName(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "PodList"}, &unstructured.UnstructuredList{})
	s.AddKnownTypeWithName(schema.GroupVersionKind{Group: "storage.k8s.io", Version: "v1", Kind: "VolumeAttachmentList"}, &unstructured.UnstructuredList{})
	return s
}

func newPVC(name, namespace, volumeName string, accessModes []string) *unstructured.Unstructured {
	modes := make([]any, 0, len(accessModes))
	for _, m := range accessModes {
		modes = append(modes, m)
	}
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "PersistentVolumeClaim",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"accessModes": modes,
				"volumeName":  volumeName,
			},
			"status": map[string]any{"phase": "Bound"},
		},
	}
}

func newVolumeAttachment(name, pvName, nodeName string) *unstructured.Unstructured {
	va := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "storage.k8s.io/v1",
			"kind":       "VolumeAttachment",
			"metadata":   map[string]any{"name": name},
			"spec": map[string]any{
				"source":   map[string]any{"persistentVolumeName": pvName},
				"nodeName": nodeName,
			},
		},
	}
	return va
}

func newRunningPodMountingPVC(name, namespace, nodeName, pvcName string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"nodeName": nodeName,
				"volumes": []any{
					map[string]any{
						"name":                  "data",
						"persistentVolumeClaim": map[string]any{"claimName": pvcName},
					},
				},
			},
			"status": map[string]any{"phase": "Running"},
		},
	}
}

func connWithObjs(objs ...runtime.Object) *cluster.Connection {
	scheme := vbTestScheme()
	dc := fakeDynamic.NewSimpleDynamicClient(scheme, objs...)
	return &cluster.Connection{Dynamic: dc}
}

func TestResolveNode_RWXShortCircuit(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadWriteMany"})
	conn := connWithObjs()
	node, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "", node)
}

func TestResolveNode_ROXShortCircuit(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadOnlyMany"})
	conn := connWithObjs()
	node, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "", node)
}

func TestResolveNode_VolumeAttachmentHit(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	va := newVolumeAttachment("va-1", "pv-1", "node-a")
	conn := connWithObjs(pvc, va)

	node, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "node-a", node)
}

func TestResolveNode_VolumeAttachmentForbidden_FallsThroughToPodScan(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	pod := newRunningPodMountingPVC("user-pod", "default", "node-b", "data")
	conn := connWithObjs(pvc, pod)

	dc := conn.Dynamic.(*fakeDynamic.FakeDynamicClient)
	dc.PrependReactor("list", "volumeattachments", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.NewForbidden(schema.GroupResource{Group: "storage.k8s.io", Resource: "volumeattachments"}, "", nil)
	})

	node, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "node-b", node)
}

func TestResolveNode_VolumeAttachmentGenericError_Propagates(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	pod := newRunningPodMountingPVC("user-pod", "default", "node-b", "data")
	conn := connWithObjs(pvc, pod)

	dc := conn.Dynamic.(*fakeDynamic.FakeDynamicClient)
	dc.PrependReactor("list", "volumeattachments", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("internal server error")
	})

	_, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNotNil(t, err)
}

func TestResolveNode_NoRunningPods_ReturnsEmpty(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	pendingPod := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]any{"name": "pending", "namespace": "default"},
			"spec": map[string]any{
				"nodeName": "node-x",
				"volumes": []any{
					map[string]any{
						"name":                  "data",
						"persistentVolumeClaim": map[string]any{"claimName": "data"},
					},
				},
			},
			"status": map[string]any{"phase": "Pending"},
		},
	}
	conn := connWithObjs(pvc, pendingPod)

	node, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "", node)
}

func TestResolveNode_DetachedVolume_ReturnsEmpty(t *testing.T) {
	pvc := newPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	// VolumeAttachment references a different PV name, pod mounts a different PVC.
	otherVA := newVolumeAttachment("va-other", "pv-other", "node-z")
	otherPod := newRunningPodMountingPVC("other", "default", "node-q", "other-pvc")
	conn := connWithObjs(pvc, otherVA, otherPod)

	node, err := ResolveNode(context.Background(), conn, pvc)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "", node)
}

func TestResolveNode_NilPVC(t *testing.T) {
	conn := connWithObjs()
	_, err := ResolveNode(context.Background(), conn, nil)
	testza.AssertNotNil(t, err)
}
