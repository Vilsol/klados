package metrics

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

type MetricsServerProvider struct {
	client metricsv1beta1.MetricsV1beta1Interface
	disc   discovery.DiscoveryInterface
}

func NewMetricsServerProvider(client metricsv1beta1.MetricsV1beta1Interface, disc discovery.DiscoveryInterface) *MetricsServerProvider {
	return &MetricsServerProvider{client: client, disc: disc}
}

func (p *MetricsServerProvider) Name() string { return "metrics-server" }

func (p *MetricsServerProvider) Available() bool {
	groups, err := p.disc.ServerGroups()
	if err != nil {
		return false
	}
	for _, g := range groups.Groups {
		if g.Name == "metrics.k8s.io" {
			return true
		}
	}
	return false
}

func (p *MetricsServerProvider) QueryRange(_ context.Context, _ string, _, _ time.Time, _ time.Duration) ([]TimeSeries, error) {
	return nil, ErrNotSupported
}

func (p *MetricsServerProvider) QueryInstant(ctx context.Context, resourceType string, namespace string, name string) (*MetricsResponse, error) {
	switch resourceType {
	case "core.v1.pods":
		return p.queryPodMetrics(ctx, namespace, name)
	case "core.v1.nodes":
		return p.queryNodeMetrics(ctx, name)
	default:
		return nil, fmt.Errorf("unsupported resource type for metrics-server: %s", resourceType)
	}
}

func (p *MetricsServerProvider) QueryNamespaceMetrics(ctx context.Context, namespace string) (*MetricsResponse, error) {
	list, err := p.client.PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pod metrics: %w", err)
	}

	var totalCPU, totalMem float64
	now := time.Now().Unix()

	for _, pm := range list.Items {
		for _, c := range pm.Containers {
			totalCPU += quantityToCores(c.Usage, corev1.ResourceCPU)
			totalMem += quantityToBytes(c.Usage, corev1.ResourceMemory)
		}
	}

	return &MetricsResponse{
		Metrics: []MetricResult{
			{
				Name: "CPU Usage",
				Unit: "cores",
				Series: []TimeSeries{{
					Labels: map[string]string{"namespace": namespace},
					Points: []TimeSeriesPoint{{Timestamp: now, Value: totalCPU}},
				}},
			},
			{
				Name: "Memory Usage",
				Unit: "bytes",
				Series: []TimeSeries{{
					Labels: map[string]string{"namespace": namespace},
					Points: []TimeSeriesPoint{{Timestamp: now, Value: totalMem}},
				}},
			},
		},
	}, nil
}

func (p *MetricsServerProvider) queryPodMetrics(ctx context.Context, namespace, name string) (*MetricsResponse, error) {
	pm, err := p.client.PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pod metrics: %w", err)
	}

	now := pm.Timestamp.Unix()
	var cpuSeries, memSeries []TimeSeries

	for _, c := range pm.Containers {
		cpu := quantityToCores(c.Usage, corev1.ResourceCPU)
		mem := quantityToBytes(c.Usage, corev1.ResourceMemory)

		labels := map[string]string{"container": c.Name}
		cpuSeries = append(cpuSeries, TimeSeries{
			Labels: labels,
			Points: []TimeSeriesPoint{{Timestamp: now, Value: cpu}},
		})
		memSeries = append(memSeries, TimeSeries{
			Labels: labels,
			Points: []TimeSeriesPoint{{Timestamp: now, Value: mem}},
		})
	}

	return &MetricsResponse{
		Metrics: []MetricResult{
			{Name: "CPU Usage", Unit: "cores", Series: cpuSeries},
			{Name: "Memory Usage", Unit: "bytes", Series: memSeries},
		},
	}, nil
}

func (p *MetricsServerProvider) queryNodeMetrics(ctx context.Context, name string) (*MetricsResponse, error) {
	nm, err := p.client.NodeMetricses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting node metrics: %w", err)
	}

	now := nm.Timestamp.Unix()
	cpu := quantityToCores(nm.Usage, corev1.ResourceCPU)
	mem := quantityToBytes(nm.Usage, corev1.ResourceMemory)

	labels := map[string]string{"node": name}
	return &MetricsResponse{
		Metrics: []MetricResult{
			{
				Name:   "CPU Usage",
				Unit:   "cores",
				Series: []TimeSeries{{Labels: labels, Points: []TimeSeriesPoint{{Timestamp: now, Value: cpu}}}},
			},
			{
				Name:   "Memory Usage",
				Unit:   "bytes",
				Series: []TimeSeries{{Labels: labels, Points: []TimeSeriesPoint{{Timestamp: now, Value: mem}}}},
			},
		},
	}, nil
}

func (p *MetricsServerProvider) ListPodMetrics(ctx context.Context, namespace string) (map[string][]MetricResult, error) {
	list, err := p.client.PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pod metrics: %w", err)
	}

	if len(list.Items) > 200 {
		return nil, ErrTooManyResources
	}

	result := make(map[string][]MetricResult, len(list.Items))
	now := time.Now().Unix()

	for _, pm := range list.Items {
		var totalCPU, totalMem float64
		for _, c := range pm.Containers {
			totalCPU += quantityToCores(c.Usage, corev1.ResourceCPU)
			totalMem += quantityToBytes(c.Usage, corev1.ResourceMemory)
		}
		result[pm.Name] = []MetricResult{
			{
				Name: "CPU",
				Unit: "cores",
				Series: []TimeSeries{{
					Labels: map[string]string{"pod": pm.Name},
					Points: []TimeSeriesPoint{{Timestamp: now, Value: totalCPU}},
				}},
			},
			{
				Name: "Memory",
				Unit: "bytes",
				Series: []TimeSeries{{
					Labels: map[string]string{"pod": pm.Name},
					Points: []TimeSeriesPoint{{Timestamp: now, Value: totalMem}},
				}},
			},
		}
	}

	return result, nil
}

func (p *MetricsServerProvider) ListNodeMetrics(ctx context.Context) (map[string][]MetricResult, error) {
	list, err := p.client.NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing node metrics: %w", err)
	}

	if len(list.Items) > 200 {
		return nil, ErrTooManyResources
	}

	result := make(map[string][]MetricResult, len(list.Items))
	now := time.Now().Unix()

	for _, nm := range list.Items {
		cpu := quantityToCores(nm.Usage, corev1.ResourceCPU)
		mem := quantityToBytes(nm.Usage, corev1.ResourceMemory)
		result[nm.Name] = []MetricResult{
			{
				Name: "CPU",
				Unit: "cores",
				Series: []TimeSeries{{
					Labels: map[string]string{"node": nm.Name},
					Points: []TimeSeriesPoint{{Timestamp: now, Value: cpu}},
				}},
			},
			{
				Name: "Memory",
				Unit: "bytes",
				Series: []TimeSeries{{
					Labels: map[string]string{"node": nm.Name},
					Points: []TimeSeriesPoint{{Timestamp: now, Value: mem}},
				}},
			},
		}
	}

	return result, nil
}

func quantityToCores(usage corev1.ResourceList, name corev1.ResourceName) float64 {
	q, ok := usage[name]
	if !ok {
		return 0
	}
	return float64(q.MilliValue()) / 1000.0
}

func quantityToBytes(usage corev1.ResourceList, name corev1.ResourceName) float64 {
	q, ok := usage[name]
	if !ok {
		return 0
	}
	return float64(q.Value())
}
