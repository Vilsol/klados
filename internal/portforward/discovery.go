package portforward

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Vilsol/klados/internal/cluster"
)

var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

// selectBestPod picks the best pod from those matching selector.
// Priority: Running > Ready > newest creationTimestamp.
func selectBestPod(ctx context.Context, conn *cluster.Connection, namespace string, selector map[string]string) (string, error) {
	labelSelector := labels.Set(selector).String()
	podList, err := conn.Dynamic.Resource(podGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("listing pods: %w", err)
	}

	type candidate struct {
		name    string
		running bool
		ready   bool
		created time.Time
	}

	var candidates []candidate
	for _, item := range podList.Items {
		var pod corev1.Pod
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &pod); err != nil {
			continue
		}
		running := pod.Status.Phase == corev1.PodRunning
		ready := false
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		candidates = append(candidates, candidate{
			name:    pod.Name,
			running: running,
			ready:   ready,
			created: pod.CreationTimestamp.Time,
		})
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no pods found matching selector %q in namespace %q", labelSelector, namespace)
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.running != b.running {
			return a.running
		}
		if a.ready != b.ready {
			return a.ready
		}
		return a.created.After(b.created)
	})

	return candidates[0].name, nil
}

// resolvePodTarget returns the pod name to forward to.
// For pod/statefulpod targets it returns TargetName directly.
// For selector targets it fetches the workload/service selector and calls selectBestPod.
func resolvePodTarget(ctx context.Context, conn *cluster.Connection, spec *ForwardSpec) (string, error) {
	switch spec.TargetKind {
	case TargetKindPod, TargetKindStatefulPod:
		return spec.TargetName, nil
	case TargetKindSelector:
		sel, err := fetchSelector(ctx, conn, spec.Namespace, spec.TargetGVR, spec.TargetName)
		if err != nil {
			return "", err
		}
		return selectBestPod(ctx, conn, spec.Namespace, sel)
	default:
		return "", fmt.Errorf("unknown target kind %q", spec.TargetKind)
	}
}

func fetchSelector(ctx context.Context, conn *cluster.Connection, namespace, gvrStr, name string) (map[string]string, error) {
	gvr, err := parseGVR(gvrStr)
	if err != nil {
		return nil, err
	}
	obj, err := conn.Dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting %s/%s: %w", gvrStr, name, err)
	}

	spec, _ := obj.Object["spec"].(map[string]interface{})
	if spec == nil {
		return nil, fmt.Errorf("no spec in %s/%s", gvrStr, name)
	}

	// Service: spec.selector
	if sel, ok := spec["selector"].(map[string]interface{}); ok {
		m := toStringMap(sel)
		if len(m) > 0 {
			return m, nil
		}
	}

	// Deployment/StatefulSet: spec.selector.matchLabels
	if selObj, ok := spec["selector"].(map[string]interface{}); ok {
		if ml, ok := selObj["matchLabels"].(map[string]interface{}); ok {
			m := toStringMap(ml)
			if len(m) > 0 {
				return m, nil
			}
		}
	}

	return nil, fmt.Errorf("no selector found in %s/%s", gvrStr, name)
}

func toStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

func parseGVR(gvrStr string) (schema.GroupVersionResource, error) {
	// Splits from the right: last segment = resource, second-to-last = version, rest = group.
	// Mirrors resource.ParseGVR logic but avoids an import cycle.
	n := len(gvrStr)
	r := lastIndex(gvrStr, '.')
	if r == -1 || r == n-1 {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid GVR %q", gvrStr)
	}
	resource := gvrStr[r+1:]
	rest := gvrStr[:r]

	v := lastIndex(rest, '.')
	if v == -1 || v == len(rest)-1 {
		return schema.GroupVersionResource{}, fmt.Errorf("invalid GVR %q", gvrStr)
	}
	version := rest[v+1:]
	group := rest[:v]
	if group == "core" {
		group = ""
	}
	return schema.GroupVersionResource{Group: group, Version: version, Resource: resource}, nil
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}
