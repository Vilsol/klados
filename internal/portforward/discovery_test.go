package portforward

import (
	"context"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/Vilsol/klados/internal/cluster"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = corev1.AddToScheme(testScheme)
}

func fakeConnWithPods(pods ...*corev1.Pod) *cluster.Connection {
	var objs []runtime.Object
	for _, p := range pods {
		objs = append(objs, p)
	}
	return &cluster.Connection{
		Clientset: fake.NewSimpleClientset(objs...),
		Dynamic:   fakeDynamic.NewSimpleDynamicClient(testScheme, objs...),
	}
}

func makePod(name string, phase corev1.PodPhase, ready bool, created time.Time) *corev1.Pod {
	readyStatus := corev1.ConditionFalse
	if ready {
		readyStatus = corev1.ConditionTrue
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			Labels:            map[string]string{"app": "test"},
			CreationTimestamp: metav1.NewTime(created),
		},
		Status: corev1.PodStatus{
			Phase: phase,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: readyStatus},
			},
		},
	}
}

func TestSelectBestPod_PrefersRunning(t *testing.T) {
	now := time.Now()
	pending := makePod("pending-pod", corev1.PodPending, false, now)
	running := makePod("running-pod", corev1.PodRunning, false, now)
	conn := fakeConnWithPods(pending, running)

	name, err := selectBestPod(context.Background(), conn, "default", map[string]string{"app": "test"})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "running-pod", name)
}

func TestSelectBestPod_PrefersReady(t *testing.T) {
	now := time.Now()
	notReady := makePod("not-ready", corev1.PodRunning, false, now)
	ready := makePod("ready", corev1.PodRunning, true, now)
	conn := fakeConnWithPods(notReady, ready)

	name, err := selectBestPod(context.Background(), conn, "default", map[string]string{"app": "test"})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "ready", name)
}

func TestSelectBestPod_TiebreakByNewest(t *testing.T) {
	older := makePod("older", corev1.PodRunning, true, time.Now().Add(-5*time.Minute))
	newer := makePod("newer", corev1.PodRunning, true, time.Now())
	conn := fakeConnWithPods(older, newer)

	name, err := selectBestPod(context.Background(), conn, "default", map[string]string{"app": "test"})
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "newer", name)
}

func TestSelectBestPod_NoPods(t *testing.T) {
	conn := fakeConnWithPods()
	_, err := selectBestPod(context.Background(), conn, "default", map[string]string{"app": "test"})
	testza.AssertNotNil(t, err)
}

func TestResolvePodTarget_RawPod(t *testing.T) {
	conn := fakeConnWithPods()
	spec := &ForwardSpec{TargetKind: TargetKindPod, TargetName: "my-pod"}
	name, err := resolvePodTarget(context.Background(), conn, spec)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "my-pod", name)
}

func TestResolvePodTarget_StatefulPod(t *testing.T) {
	conn := fakeConnWithPods()
	spec := &ForwardSpec{TargetKind: TargetKindStatefulPod, TargetName: "my-sts-0"}
	name, err := resolvePodTarget(context.Background(), conn, spec)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "my-sts-0", name)
}

func TestResolvePodTarget_Selector(t *testing.T) {
	now := time.Now()
	pod := makePod("ready-pod", corev1.PodRunning, true, now)

	svc := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata":   map[string]interface{}{"name": "my-svc", "namespace": "default"},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{"app": "test"},
			},
		},
	}
	svcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	conn := fakeConnWithPods(pod)
	_, _ = conn.Dynamic.Resource(svcGVR).Namespace("default").Create(context.Background(), svc, metav1.CreateOptions{})

	spec := &ForwardSpec{
		TargetKind: TargetKindSelector,
		TargetName: "my-svc",
		TargetGVR:  "core.v1.services",
		Namespace:  "default",
	}
	name, err := resolvePodTarget(context.Background(), conn, spec)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "ready-pod", name)
}

func TestParseGVR(t *testing.T) {
	tests := []struct {
		input   string
		group   string
		version string
		res     string
		wantErr bool
	}{
		{"core.v1.services", "", "v1", "services", false},
		{"apps.v1.deployments", "apps", "v1", "deployments", false},
		{"networking.k8s.io.v1.ingresses", "networking.k8s.io", "v1", "ingresses", false},
		{"invalid", "", "", "", true},
		{"a.b", "", "", "", true},
	}
	for _, tc := range tests {
		gvr, err := parseGVR(tc.input)
		if tc.wantErr {
			testza.AssertNotNil(t, err)
			continue
		}
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, tc.group, gvr.Group)
		testza.AssertEqual(t, tc.version, gvr.Version)
		testza.AssertEqual(t, tc.res, gvr.Resource)
	}
}
