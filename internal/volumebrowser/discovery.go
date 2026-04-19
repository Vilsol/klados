package volumebrowser

import (
	"context"
	"fmt"

	"github.com/Vilsol/slox"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Vilsol/klados/internal/cluster"
)

var (
	pvcGVR              = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
	podGVR              = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	volumeAttachmentGVR = schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "volumeattachments"}
)

// ResolveNode determines the node a PVC is attached to.
//
// Rules:
//  1. If access modes include ReadWriteMany or ReadOnlyMany, return "" (any node works).
//  2. Else list VolumeAttachment (cluster-scoped) and match on spec.source.persistentVolumeName.
//     On RBAC forbidden: log debug and fall through.
//  3. Else list pods in pvc.namespace, find a Running pod that mounts the PVC, return its nodeName.
//  4. Otherwise return "".
func ResolveNode(ctx context.Context, conn *cluster.Connection, pvc *unstructured.Unstructured) (string, error) {
	if pvc == nil {
		return "", fmt.Errorf("pvc is nil")
	}

	// 1. RWX/ROX short-circuit.
	accessModes, _, _ := unstructured.NestedStringSlice(pvc.Object, "spec", "accessModes")
	for _, m := range accessModes {
		if m == "ReadWriteMany" || m == "ReadOnlyMany" {
			return "", nil
		}
	}

	volumeName, _, _ := unstructured.NestedString(pvc.Object, "spec", "volumeName")

	// 2. VolumeAttachment lookup (only useful if we have a bound PV name).
	if volumeName != "" {
		vaList, err := conn.Dynamic.Resource(volumeAttachmentGVR).List(ctx, metav1.ListOptions{})
		switch {
		case err == nil:
			for _, va := range vaList.Items {
				pvName, _, _ := unstructured.NestedString(va.Object, "spec", "source", "persistentVolumeName")
				if pvName != volumeName {
					continue
				}
				nodeName, _, _ := unstructured.NestedString(va.Object, "spec", "nodeName")
				if nodeName != "" {
					return nodeName, nil
				}
			}
		case apierrors.IsForbidden(err):
			slox.Debug(ctx, "volumebrowser: VolumeAttachment list forbidden, falling back to pod-scan", "error", err)
		default:
			return "", fmt.Errorf("list VolumeAttachments: %w", err)
		}
	}

	// 3. Pod-scan fallback.
	podList, err := conn.Dynamic.Resource(podGVR).Namespace(pvc.GetNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("listing pods in namespace %q: %w", pvc.GetNamespace(), err)
	}
	for _, pod := range podList.Items {
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		if phase != "Running" {
			continue
		}
		volumes, _, _ := unstructured.NestedSlice(pod.Object, "spec", "volumes")
		if !volumesReferencePVC(volumes, pvc.GetName()) {
			continue
		}
		nodeName, _, _ := unstructured.NestedString(pod.Object, "spec", "nodeName")
		if nodeName != "" {
			return nodeName, nil
		}
	}
	return "", nil
}

func volumesReferencePVC(volumes []any, pvcName string) bool {
	for _, raw := range volumes {
		v, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		pvcRef, ok := v["persistentVolumeClaim"].(map[string]any)
		if !ok {
			continue
		}
		claimName, _ := pvcRef["claimName"].(string)
		if claimName == pvcName {
			return true
		}
	}
	return false
}
