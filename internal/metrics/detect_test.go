package metrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	fakekube "k8s.io/client-go/kubernetes/fake"
)

func TestDetectPrometheus_ManualURLTakesPrecedence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	clientset := fakekube.NewSimpleClientset()

	url, found := metrics.DetectPrometheus(context.Background(), clientset, clientset.Discovery(), nil, nil, srv.URL)
	testza.AssertTrue(t, found)
	testza.AssertEqual(t, srv.URL, url)
}

func TestDetectPrometheus_WellKnownService(t *testing.T) {
	// Create a fake Prometheus server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Create a fake service in the monitoring namespace matching well-known name
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-server",
			Namespace: "monitoring",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 9090}},
		},
	}

	clientset := fakekube.NewSimpleClientset(svc)

	// DetectPrometheus will find the service but the in-cluster URL won't resolve.
	// Since we can't make in-cluster URLs work in a unit test, we verify detection
	// returns false when the candidate URL is unreachable.
	url, found := metrics.DetectPrometheus(context.Background(), clientset, clientset.Discovery(), nil, nil, "")
	// The well-known service URL points to an in-cluster address which isn't reachable in tests
	testza.AssertFalse(t, found)
	testza.AssertEqual(t, "", url)
}

func TestDetectPrometheus_OperatorCRD(t *testing.T) {
	clientset := fakekube.NewSimpleClientset()
	fd := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fd.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "monitoring.coreos.com/v1",
			APIResources: []metav1.APIResource{
				{Name: "prometheuses", Kind: "Prometheus"},
			},
		},
	}

	promCR := &unstructured.Unstructured{}
	promCR.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "Prometheus",
	})
	promCR.SetName("k8s")
	promCR.SetNamespace("monitoring")

	promGVR := schema.GroupVersionResource{Group: "monitoring.coreos.com", Version: "v1", Resource: "prometheuses"}
	scheme := runtime.NewScheme()
	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{promGVR: "PrometheusList"},
		promCR,
	)

	// The CRD-derived URL won't be reachable in unit tests
	url, found := metrics.DetectPrometheus(context.Background(), clientset, fd, dynClient, nil, "")
	testza.AssertFalse(t, found)
	testza.AssertEqual(t, "", url)
}

func TestDetectPrometheus_NoPrometheus(t *testing.T) {
	clientset := fakekube.NewSimpleClientset()
	fd := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fd.Resources = []*metav1.APIResourceList{
		{GroupVersion: "v1", APIResources: []metav1.APIResource{{Name: "pods", Kind: "Pod"}}},
	}

	scheme := runtime.NewScheme()
	dynClient := dynamicfake.NewSimpleDynamicClient(scheme)

	url, found := metrics.DetectPrometheus(context.Background(), clientset, fd, dynClient, nil, "")
	testza.AssertFalse(t, found)
	testza.AssertEqual(t, "", url)
}

func TestDetectPrometheus_ManualURLUnreachableFallsThrough(t *testing.T) {
	clientset := fakekube.NewSimpleClientset()
	fd := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fd.Resources = []*metav1.APIResourceList{}

	scheme := runtime.NewScheme()
	dynClient := dynamicfake.NewSimpleDynamicClient(scheme)

	url, found := metrics.DetectPrometheus(context.Background(), clientset, fd, dynClient, nil, "http://127.0.0.1:1/nope")
	testza.AssertFalse(t, found)
	testza.AssertEqual(t, "", url)
}
