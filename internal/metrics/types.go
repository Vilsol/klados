package metrics

import "errors"

var ErrNotSupported = errors.New("not supported by this provider")

var ErrTooManyResources = errors.New("too many resources for sparkline query (>200)")

// TimeSeriesPoint is a single (timestamp, value) sample.
type TimeSeriesPoint struct {
	Timestamp int64   `json:"t"`
	Value     float64 `json:"v"`
}

// TimeSeries is a labeled series of points.
type TimeSeries struct {
	Labels map[string]string `json:"labels"`
	Points []TimeSeriesPoint `json:"points"`
}

// MetricResult is the response for a single metric query.
type MetricResult struct {
	Name   string       `json:"name"`
	Unit   string       `json:"unit"`
	Series []TimeSeries `json:"series"`
}

// ThresholdLine is a horizontal overlay (requests/limits).
type ThresholdLine struct {
	Label  string            `json:"label"`
	Series []TimeSeriesPoint `json:"series"`
}

// Annotation is a vertical event marker on the graph.
type Annotation struct {
	Timestamp int64  `json:"t"`
	Label     string `json:"label"`
	Severity  string `json:"severity"`
}

// MetricsResponse is the full response for a resource's metrics tab.
type MetricsResponse struct {
	Metrics     []MetricResult  `json:"metrics"`
	Thresholds  []ThresholdLine `json:"thresholds"`
	Annotations []Annotation    `json:"annotations"`
}

// MetricQuery is a PromQL template registered by built-in code or plugins.
type MetricQuery struct {
	Name   string            `json:"name"`
	Query  string            `json:"query"`
	Unit   string            `json:"unit"`
	Vars   map[string]string `json:"vars"`
	Source string            `json:"source"`
}

// MetricsCapability describes what metric sources are available for a cluster.
type MetricsCapability struct {
	HasMetricsServer bool   `json:"hasMetricsServer"`
	HasPrometheus    bool   `json:"hasPrometheus"`
	PrometheusURL    string `json:"prometheusUrl,omitempty"`
	HasKSM           bool   `json:"hasKsm"` // kube-state-metrics is available
}
