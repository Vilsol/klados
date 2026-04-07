package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Vilsol/slox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	wellKnownNamespaces = []string{"monitoring", "prometheus", "observability", "default"}
	wellKnownServices   = []string{"prometheus", "prometheus-server", "prometheus-kube-prometheus-prometheus", "prometheus-operated"}
)

// serviceAccessor creates a PrometheusProvider that can reach a cluster service
// identified by namespace, name, and port.
//
// Today this is implemented via the Kubernetes API server proxy (Option A).
// To switch to port-forward tunnelling (Option B), replace APIProxyAccessor
// with a new constructor that opens a port-forward and returns localhost:{port}.
type serviceAccessor func(ctx context.Context, ns, name string, port int) (*PrometheusProvider, error)

// APIProxyAccessor returns a serviceAccessor that routes requests through the
// Kubernetes API server proxy endpoint:
//
//	{apiserver}/api/v1/namespaces/{ns}/services/http:{name}:{port}/proxy
//
// This works from outside the cluster because all traffic is authenticated and
// tunnelled through the API server using the existing rest.Config transport.
func APIProxyAccessor(restConfig *rest.Config) serviceAccessor {
	return func(_ context.Context, ns, name string, port int) (*PrometheusProvider, error) {
		host := strings.TrimRight(restConfig.Host, "/")
		proxyURL := fmt.Sprintf("%s/api/v1/namespaces/%s/services/http:%s:%d/proxy", host, ns, name, port)
		return NewPrometheusProvider(proxyURL, restConfig)
	}
}

// DetectPrometheus attempts to find a reachable Prometheus endpoint.
// Priority: manual URL > well-known services > Prometheus Operator CRDs.
// Returns the URL and whether detection succeeded.
func DetectPrometheus(
	ctx context.Context,
	clientset kubernetes.Interface,
	disc discovery.DiscoveryInterface,
	dyn dynamic.Interface,
	restConfig *rest.Config,
	manualURL string,
) (string, bool) {
	slox.Debug(ctx, "starting Prometheus detection")

	if manualURL != "" {
		slox.Debug(ctx, "trying manually configured Prometheus endpoint", "url", manualURL)
		provider, err := NewPrometheusProvider(manualURL, nil)
		if err == nil {
			if probeErr := provider.probe(); probeErr == nil {
				slox.Info(ctx, "using manually configured Prometheus endpoint", "url", manualURL)
				return manualURL, true
			} else {
				slox.Warn(ctx, "manual Prometheus endpoint not reachable", "url", manualURL, "reason", probeErr)
			}
		}
	} else {
		slox.Debug(ctx, "no manual Prometheus URL configured")
	}

	if restConfig == nil {
		slox.Debug(ctx, "no rest config available — skipping in-cluster service discovery")
	} else {
		accessor := APIProxyAccessor(restConfig)

		slox.Debug(ctx, "trying well-known service names", "namespaces", wellKnownNamespaces, "services", wellKnownServices)
		if url, ok := detectWellKnownServices(ctx, clientset, accessor); ok {
			return url, true
		}
		slox.Debug(ctx, "no well-known Prometheus service found")

		slox.Debug(ctx, "trying Prometheus Operator CRD discovery")
		if url, ok := detectPrometheusOperator(ctx, disc, dyn, accessor); ok {
			return url, true
		}
		slox.Debug(ctx, "no Prometheus Operator CR found")
	}

	slox.Debug(ctx, "Prometheus detection exhausted — no endpoint found")
	return "", false
}

func detectWellKnownServices(ctx context.Context, clientset kubernetes.Interface, access serviceAccessor) (string, bool) {
	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for _, ns := range wellKnownNamespaces {
		svcList, err := clientset.CoreV1().Services(ns).List(listCtx, metav1.ListOptions{})
		if err != nil {
			slox.Debug(ctx, "could not list services", "namespace", ns, "error", err)
			continue
		}
		slox.Debug(ctx, "listed services in namespace", "namespace", ns, "count", len(svcList.Items))
		for _, svc := range svcList.Items {
			for _, wk := range wellKnownServices {
				if svc.Name == wk {
					provider, err := access(ctx, ns, svc.Name, 9090)
					if err != nil {
						slox.Debug(ctx, "failed to create Prometheus provider for candidate", "service", svc.Name, "namespace", ns, "error", err)
						continue
					}
					slox.Debug(ctx, "found well-known service name, probing endpoint", "service", svc.Name, "namespace", ns, "url", provider.baseURL)
					if probeErr := provider.probe(); probeErr != nil {
						slox.Debug(ctx, "well-known service endpoint not reachable", "url", provider.baseURL, "reason", probeErr)
						continue
					}
					slox.Info(ctx, "auto-detected Prometheus via well-known service", "service", svc.Name, "namespace", ns, "url", provider.baseURL)
					return provider.baseURL, true
				}
			}
		}
	}
	return "", false
}

func detectPrometheusOperator(ctx context.Context, disc discovery.DiscoveryInterface, dyn dynamic.Interface, access serviceAccessor) (string, bool) {
	groups, err := disc.ServerGroups()
	if err != nil {
		slox.Debug(ctx, "could not fetch API groups for Operator CRD check", "error", err)
		return "", false
	}

	hasMonitoringAPI := false
	for _, g := range groups.Groups {
		if g.Name == "monitoring.coreos.com" {
			hasMonitoringAPI = true
			break
		}
	}
	if !hasMonitoringAPI {
		slox.Debug(ctx, "monitoring.coreos.com API group not present — Prometheus Operator not installed")
		return "", false
	}
	slox.Debug(ctx, "monitoring.coreos.com API group found — listing Prometheus CRs")

	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	promGVR := schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "prometheuses",
	}

	list, err := dyn.Resource(promGVR).Namespace("").List(listCtx, metav1.ListOptions{})
	if err != nil {
		slox.Warn(ctx, "failed to list Prometheus CRs", "error", err)
		return "", false
	}
	slox.Debug(ctx, "found Prometheus CRs", "count", len(list.Items))

	for _, item := range list.Items {
		name := item.GetName()
		ns := item.GetNamespace()
		// Prometheus Operator names the service "prometheus-{cr-name}" or just the CR name.
		candidateNames := []string{"prometheus-" + name, name}
		slox.Debug(ctx, "probing Prometheus CR service candidates", "cr", name, "namespace", ns, "candidates", candidateNames)
		for _, svcName := range candidateNames {
			provider, err := access(ctx, ns, svcName, 9090)
			if err != nil {
				slox.Debug(ctx, "failed to create Prometheus provider for CR candidate", "service", svcName, "namespace", ns, "error", err)
				continue
			}
			slox.Debug(ctx, "probing CR candidate", "url", provider.baseURL)
			if probeErr := provider.probe(); probeErr != nil {
				slox.Debug(ctx, "Prometheus CR candidate endpoint not reachable", "url", provider.baseURL, "reason", probeErr)
				continue
			}
			slox.Info(ctx, "auto-detected Prometheus via Operator CR", "name", name, "namespace", ns, "url", provider.baseURL)
			return provider.baseURL, true
		}
	}

	return "", false
}
