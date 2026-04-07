package metrics

import (
	"context"
	"time"
)

// MetricsProvider is the abstraction over metric data sources.
type MetricsProvider interface {
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]TimeSeries, error)
	QueryInstant(ctx context.Context, resourceType string, namespace string, name string) (*MetricsResponse, error)
	Available() bool
	Name() string
}

type providerSet struct {
	metricsServer MetricsProvider
	prometheus    MetricsProvider
}
