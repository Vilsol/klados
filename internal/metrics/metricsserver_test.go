package metrics_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakekube "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	fakemetrics "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func fakeDiscoveryWithMetricsAPI() *fakediscovery.FakeDiscovery {
	client := fakekube.NewSimpleClientset()
	fd := client.Discovery().(*fakediscovery.FakeDiscovery)
	fd.FakedServerVersion = nil
	fd.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "metrics.k8s.io/v1beta1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "PodMetrics"},
				{Name: "nodes", Kind: "NodeMetrics"},
			},
		},
	}
	return fd
}

func fakeDiscoveryWithoutMetricsAPI() *fakediscovery.FakeDiscovery {
	client := fakekube.NewSimpleClientset()
	fd := client.Discovery().(*fakediscovery.FakeDiscovery)
	fd.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Kind: "Pod"},
			},
		},
	}
	return fd
}

func fakeMetricsWithPodGet(pm *metricsv1beta1.PodMetrics) *fakemetrics.Clientset {
	mc := fakemetrics.NewSimpleClientset()
	mc.PrependReactor("get", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, pm, nil
	})
	return mc
}

func fakeMetricsWithPodList(pods ...*metricsv1beta1.PodMetrics) *fakemetrics.Clientset {
	mc := fakemetrics.NewSimpleClientset()
	mc.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		list := &metricsv1beta1.PodMetricsList{}
		for _, p := range pods {
			list.Items = append(list.Items, *p)
		}
		return true, list, nil
	})
	return mc
}

func fakeMetricsWithNodeGet(nm *metricsv1beta1.NodeMetrics) *fakemetrics.Clientset {
	mc := fakemetrics.NewSimpleClientset()
	mc.PrependReactor("get", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nm, nil
	})
	return mc
}

func TestAvailable_WithMetricsAPI(t *testing.T) {
	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakemetrics.NewSimpleClientset()
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)
	testza.AssertTrue(t, provider.Available())
}

func TestAvailable_WithoutMetricsAPI(t *testing.T) {
	disc := fakeDiscoveryWithoutMetricsAPI()
	mc := fakemetrics.NewSimpleClientset()
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)
	testza.AssertFalse(t, provider.Available())
}

func TestQueryRange_ReturnsErrNotSupported(t *testing.T) {
	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakemetrics.NewSimpleClientset()
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	_, err := provider.QueryRange(context.Background(), "query", metav1.Now().Time, metav1.Now().Time, 0)
	testza.AssertEqual(t, metrics.ErrNotSupported, err)
}

func TestQueryInstant_Pod_NormalizesNanocores(t *testing.T) {
	pm := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Timestamp:  metav1.Now(),
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithPodGet(pm)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	resp, err := provider.QueryInstant(context.Background(), "core.v1.pods", "default", "test-pod")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, resp.Metrics, 2)

	cpuResult := resp.Metrics[0]
	testza.AssertEqual(t, "CPU Usage", cpuResult.Name)
	testza.AssertEqual(t, "cores", cpuResult.Unit)
	testza.AssertEqual(t, 0.5, cpuResult.Series[0].Points[0].Value)

	memResult := resp.Metrics[1]
	testza.AssertEqual(t, "Memory Usage", memResult.Name)
	testza.AssertEqual(t, "bytes", memResult.Unit)
	testza.AssertEqual(t, float64(128*1024*1024), memResult.Series[0].Points[0].Value)
}

func TestQueryInstant_Pod_SeriesPerContainer(t *testing.T) {
	pm := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "multi-pod", Namespace: "default"},
		Timestamp:  metav1.Now(),
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "nginx",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
			{
				Name: "sidecar",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
			},
		},
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithPodGet(pm)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	resp, err := provider.QueryInstant(context.Background(), "core.v1.pods", "default", "multi-pod")
	testza.AssertNoError(t, err)

	cpuResult := resp.Metrics[0]
	testza.AssertLen(t, cpuResult.Series, 2)
	testza.AssertEqual(t, "nginx", cpuResult.Series[0].Labels["container"])
	testza.AssertEqual(t, "sidecar", cpuResult.Series[1].Labels["container"])
}

func TestQueryInstant_Node(t *testing.T) {
	nm := &metricsv1beta1.NodeMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Timestamp:  metav1.Now(),
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithNodeGet(nm)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	resp, err := provider.QueryInstant(context.Background(), "core.v1.nodes", "", "node-1")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, resp.Metrics, 2)

	testza.AssertEqual(t, 2.0, resp.Metrics[0].Series[0].Points[0].Value)
	testza.AssertEqual(t, float64(4*1024*1024*1024), resp.Metrics[1].Series[0].Points[0].Value)
}

func TestQueryNamespaceMetrics_AggregatesAcrossPods(t *testing.T) {
	pod1 := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"},
		Timestamp:  metav1.Now(),
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
		},
	}
	pod2 := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default"},
		Timestamp:  metav1.Now(),
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			},
		},
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithPodList(pod1, pod2)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	resp, err := provider.QueryNamespaceMetrics(context.Background(), "default")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, resp.Metrics, 2)

	testza.AssertEqual(t, 0.5, resp.Metrics[0].Series[0].Points[0].Value)
	testza.AssertEqual(t, float64(300*1024*1024), resp.Metrics[1].Series[0].Points[0].Value)
}

func fakeMetricsWithNodeList(nodes ...*metricsv1beta1.NodeMetrics) *fakemetrics.Clientset {
	mc := fakemetrics.NewSimpleClientset()
	mc.PrependReactor("list", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		list := &metricsv1beta1.NodeMetricsList{}
		for _, n := range nodes {
			list.Items = append(list.Items, *n)
		}
		return true, list, nil
	})
	return mc
}

func TestListPodMetrics_KeysByPodName(t *testing.T) {
	pod1 := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "default"},
		Timestamp:  metav1.Now(),
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
		},
	}
	pod2 := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "web-2", Namespace: "default"},
		Timestamp:  metav1.Now(),
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithPodList(pod1, pod2)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	result, err := provider.ListPodMetrics(context.Background(), "default")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, result, 2)

	web1 := result["web-1"]
	testza.AssertLen(t, web1, 2)
	testza.AssertEqual(t, "CPU", web1[0].Name)
	testza.AssertEqual(t, 0.1, web1[0].Series[0].Points[0].Value)
	testza.AssertEqual(t, "Memory", web1[1].Name)
	testza.AssertEqual(t, float64(64*1024*1024), web1[1].Series[0].Points[0].Value)

	web2 := result["web-2"]
	testza.AssertLen(t, web2, 2)
	testza.AssertEqual(t, 0.25, web2[0].Series[0].Points[0].Value)
}

func TestListPodMetrics_TooManyResources(t *testing.T) {
	// Create 201 pods
	var pods []*metricsv1beta1.PodMetrics
	for i := 0; i < 201; i++ {
		pods = append(pods, &metricsv1beta1.PodMetrics{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("pod-%d", i), Namespace: "default"},
			Timestamp:  metav1.Now(),
			Containers: []metricsv1beta1.ContainerMetrics{
				{
					Name: "app",
					Usage: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("10m"),
						corev1.ResourceMemory: resource.MustParse("1Mi"),
					},
				},
			},
		})
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithPodList(pods...)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	_, err := provider.ListPodMetrics(context.Background(), "default")
	testza.AssertEqual(t, metrics.ErrTooManyResources, err)
}

func TestListNodeMetrics_KeysByNodeName(t *testing.T) {
	node1 := &metricsv1beta1.NodeMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
		Timestamp:  metav1.Now(),
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}
	node2 := &metricsv1beta1.NodeMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "node-b"},
		Timestamp:  metav1.Now(),
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1500m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}

	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithNodeList(node1, node2)
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	result, err := provider.ListNodeMetrics(context.Background())
	testza.AssertNoError(t, err)
	testza.AssertLen(t, result, 2)

	nodeA := result["node-a"]
	testza.AssertLen(t, nodeA, 2)
	testza.AssertEqual(t, "CPU", nodeA[0].Name)
	testza.AssertEqual(t, 2.0, nodeA[0].Series[0].Points[0].Value)
	testza.AssertEqual(t, "Memory", nodeA[1].Name)
	testza.AssertEqual(t, float64(4*1024*1024*1024), nodeA[1].Series[0].Points[0].Value)

	nodeB := result["node-b"]
	testza.AssertEqual(t, 1.5, nodeB[0].Series[0].Points[0].Value)
}

func TestListPodMetrics_EmptyNamespace(t *testing.T) {
	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakeMetricsWithPodList() // no pods
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	result, err := provider.ListPodMetrics(context.Background(), "empty-ns")
	testza.AssertNoError(t, err)
	testza.AssertLen(t, result, 0)
}

func TestGetCapabilities_NoPrometheus(t *testing.T) {
	disc := fakeDiscoveryWithMetricsAPI()
	mc := fakemetrics.NewSimpleClientset()
	provider := metrics.NewMetricsServerProvider(mc.MetricsV1beta1(), disc)

	cap := metrics.MetricsCapability{
		HasMetricsServer: provider.Available(),
	}
	testza.AssertTrue(t, cap.HasMetricsServer)
	testza.AssertFalse(t, cap.HasPrometheus)
	testza.AssertEqual(t, "", cap.PrometheusURL)
}
