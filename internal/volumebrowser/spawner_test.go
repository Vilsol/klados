package volumebrowser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Vilsol/klados/internal/config"
)

func boundPVC(name, namespace, volumeName string, accessModes []string) *unstructured.Unstructured {
	pvc := newPVC(name, namespace, volumeName, accessModes)
	// newPVC already sets status.phase=Bound
	return pvc
}

func unboundPVC(name, namespace string) *unstructured.Unstructured {
	pvc := newPVC(name, namespace, "", []string{"ReadWriteOnce"})
	_ = unstructured.SetNestedField(pvc.Object, "Pending", "status", "phase")
	return pvc
}

func TestSpawner_PodNamePattern(t *testing.T) {
	pvc := boundPVC("mydata", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")

	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "mydata"},
		Resolved: config.VolumeBrowserConfig{Image: "alpine:edge", MountPath: "/mnt/volume"},
	})
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, strings.HasPrefix(pod.PodName, "klados-pvc-mydata-"))
	// Pod name suffix: 8-char hex (4 bytes hex-encoded)
	suffix := strings.TrimPrefix(pod.PodName, "klados-pvc-mydata-")
	testza.AssertEqual(t, 8, len(suffix))
}

func TestSpawner_TruncatesLongPVCName(t *testing.T) {
	long := strings.Repeat("a", 60)
	pvc := boundPVC(long, "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: long},
		Resolved: config.VolumeBrowserConfig{},
	})
	testza.AssertNoError(t, err)
	// klados-pvc- (11) + 40 truncated + - (1) + 8 = 60
	testza.AssertEqual(t, 11+40+1+8, len(pod.PodName))
}

func TestSpawner_AllLabelsPresent(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-xyz")
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{},
	})
	testza.AssertNoError(t, err)

	created, err := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	testza.AssertNoError(t, err)
	labels := created.GetLabels()
	testza.AssertEqual(t, ManagedByValue, labels[LabelManagedBy])
	testza.AssertEqual(t, PurposeValue, labels[LabelPurpose])
	testza.AssertEqual(t, "data", labels[LabelPVC])
	testza.AssertEqual(t, "session-xyz", labels[LabelSession])
}

func TestSpawner_NilResourcesOmitsResourcesBlock(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{}, // Resources nil
	})
	testza.AssertNoError(t, err)

	created, err := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	testza.AssertNoError(t, err)
	containers, _, _ := unstructured.NestedSlice(created.Object, "spec", "containers")
	c := containers[0].(map[string]any)
	_, hasResources := c["resources"]
	testza.AssertFalse(t, hasResources)
}

func TestSpawner_NonNilResourcesPopulatesRequestsAndLimits(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	res := &config.ResourceReqs{
		Requests: map[string]string{"cpu": "10m", "memory": "16Mi"},
		Limits:   map[string]string{"cpu": "100m", "memory": "64Mi"},
	}
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{Resources: res},
	})
	testza.AssertNoError(t, err)

	created, _ := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	containers, _, _ := unstructured.NestedSlice(created.Object, "spec", "containers")
	c := containers[0].(map[string]any)
	resources, ok := c["resources"].(map[string]any)
	testza.AssertTrue(t, ok)
	reqs := resources["requests"].(map[string]any)
	lim := resources["limits"].(map[string]any)
	testza.AssertEqual(t, "10m", reqs["cpu"])
	testza.AssertEqual(t, "64Mi", lim["memory"])
}

func TestSpawner_ReadOnlyRespected(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	ro := true
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{ReadOnly: &ro},
	})
	testza.AssertNoError(t, err)

	created, _ := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	containers, _, _ := unstructured.NestedSlice(created.Object, "spec", "containers")
	c := containers[0].(map[string]any)
	vms := c["volumeMounts"].([]any)
	vm := vms[0].(map[string]any)
	testza.AssertEqual(t, true, vm["readOnly"])

	volumes, _, _ := unstructured.NestedSlice(created.Object, "spec", "volumes")
	vol := volumes[0].(map[string]any)
	pvcRef := vol["persistentVolumeClaim"].(map[string]any)
	testza.AssertEqual(t, true, pvcRef["readOnly"])
}

func TestSpawner_NodeNamePopulatedFromDiscovery(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	va := newVolumeAttachment("va-1", "pv-1", "node-a")
	conn := connWithObjs(pvc, va)
	sp := NewSpawner("session-1")
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{},
	})
	testza.AssertNoError(t, err)

	created, _ := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	nodeName, _, _ := unstructured.NestedString(created.Object, "spec", "nodeName")
	testza.AssertEqual(t, "node-a", nodeName)
}

func TestSpawner_RWXOmitsNodeName(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteMany"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{},
	})
	testza.AssertNoError(t, err)

	created, _ := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	_, hasNode, _ := unstructured.NestedString(created.Object, "spec", "nodeName")
	testza.AssertFalse(t, hasNode)
}

func TestSpawner_PVCNotBoundReturnsTypedError(t *testing.T) {
	pvc := unboundPVC("data", "default")
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	_, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{},
	})
	testza.AssertNotNil(t, err)
	testza.AssertTrue(t, errors.Is(err, ErrPVCNotBound))
}

func TestSpawner_PodSpecDefaults(t *testing.T) {
	pvc := boundPVC("data", "default", "pv-1", []string{"ReadWriteOnce"})
	conn := connWithObjs(pvc)
	sp := NewSpawner("session-1")
	deadline := int64(3600)
	pod, err := sp.Spawn(context.Background(), conn, SpawnParams{
		Request:  SpawnRequest{ContextName: "ctx1", Namespace: "default", PVCName: "data"},
		Resolved: config.VolumeBrowserConfig{ActiveDeadlineSeconds: &deadline},
	})
	testza.AssertNoError(t, err)

	created, _ := conn.Dynamic.Resource(podGVR).Namespace("default").Get(context.Background(), pod.PodName, metav1.GetOptions{})
	restart, _, _ := unstructured.NestedString(created.Object, "spec", "restartPolicy")
	testza.AssertEqual(t, "Never", restart)

	grace, _, _ := unstructured.NestedInt64(created.Object, "spec", "terminationGracePeriodSeconds")
	testza.AssertEqual(t, int64(1), grace)

	got, _, _ := unstructured.NestedInt64(created.Object, "spec", "activeDeadlineSeconds")
	testza.AssertEqual(t, int64(3600), got)

	containers, _, _ := unstructured.NestedSlice(created.Object, "spec", "containers")
	c := containers[0].(map[string]any)
	testza.AssertEqual(t, "browser", c["name"])
	cmd := c["command"].([]any)
	testza.AssertEqual(t, "sh", cmd[0])
}
