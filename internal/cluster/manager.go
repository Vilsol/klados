package cluster

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/slox"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnecting
	StatusConnected
	StatusError
)

func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

type KubeContext struct {
	Name          string           `json:"name"`
	Cluster       string           `json:"cluster"`
	User          string           `json:"user"`
	Namespace     string           `json:"namespace"`
	Status        ConnectionStatus `json:"status"`
	ServerVersion string           `json:"serverVersion"`
	Provider      string           `json:"provider"`
}

func detectProvider(clusterName, serverVersion string) string {
	switch {
	case strings.HasPrefix(clusterName, "gke_"):
		return "GKE"
	case strings.HasPrefix(clusterName, "arn:aws:eks"):
		return "EKS"
	case strings.Contains(clusterName, "minikube"):
		return "minikube"
	case strings.Contains(clusterName, "kind-"):
		return "kind"
	case strings.Contains(serverVersion, "k3s"):
		return "k3s"
	default:
		return ""
	}
}

type Connection struct {
	KubeContext
	Config    *rest.Config
	Clientset kubernetes.Interface
	Dynamic   dynamic.Interface
	Discovery discovery.DiscoveryInterface
	cancel    context.CancelFunc
}

type Manager struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	contexts    []KubeContext
	rawConfig   *clientcmdapi.Config
	emitEvent   func(string, any)
	config      *config.Config
	ctx         context.Context
}

func NewManager(emitEvent func(string, any), cfg *config.Config, ctx context.Context) *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
		emitEvent:   emitEvent,
		config:      cfg,
		ctx:         ctx,
	}
}

func (m *Manager) LoadKubeconfigs(extraPaths []string) error {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()

	if len(extraPaths) > 0 {
		rules.Precedence = append(rules.Precedence, extraPaths...)
	}

	cfg, err := rules.Load()
	if err != nil {
		return fmt.Errorf("loading kubeconfigs: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.rawConfig = cfg
	m.contexts = make([]KubeContext, 0, len(cfg.Contexts))

	for name, ctx := range cfg.Contexts {
		ns := ctx.Namespace
		if ns == "" {
			ns = "default"
		}
		kc := KubeContext{
			Name:      name,
			Cluster:   ctx.Cluster,
			User:      ctx.AuthInfo,
			Namespace: ns,
			Status:    StatusDisconnected,
		}
		if conn, ok := m.connections[name]; ok {
			kc.Status = conn.Status
		}
		m.contexts = append(m.contexts, kc)
	}

	slox.Info(m.ctx, "loaded kubeconfigs", "contexts", len(m.contexts))
	return nil
}

func (m *Manager) ListContexts() []KubeContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]KubeContext, len(m.contexts))
	copy(out, m.contexts)

	for i := range out {
		if conn, ok := m.connections[out[i].Name]; ok {
			out[i].Status = conn.Status
			out[i].ServerVersion = conn.ServerVersion
			out[i].Provider = conn.Provider
		}
	}

	return out
}

func (m *Manager) Connect(ctx context.Context, contextName string) error {
	m.mu.Lock()
	if _, ok := m.connections[contextName]; ok {
		m.mu.Unlock()
		return nil
	}

	if m.rawConfig == nil {
		m.mu.Unlock()
		return fmt.Errorf("kubeconfigs not loaded")
	}

	rawCtx, ok := m.rawConfig.Contexts[contextName]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("context %q not found", contextName)
	}
	m.mu.Unlock()

	m.emitStatus(contextName, StatusConnecting)

	clientCfg := clientcmd.NewDefaultClientConfig(*m.rawConfig, &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	})

	restCfg, err := clientCfg.ClientConfig()
	if err != nil {
		m.emitStatus(contextName, StatusError)
		return fmt.Errorf("building rest config for %q: %w", contextName, err)
	}

	if m.config != nil && m.config.InsecureSkipTLSVerify {
		restCfg.TLSClientConfig.Insecure = true
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		m.emitStatus(contextName, StatusError)
		return fmt.Errorf("creating clientset for %q: %w", contextName, err)
	}

	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		m.emitStatus(contextName, StatusError)
		return fmt.Errorf("creating dynamic client for %q: %w", contextName, err)
	}

	disc, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		m.emitStatus(contextName, StatusError)
		return fmt.Errorf("creating discovery client for %q: %w", contextName, err)
	}

	connCtx, cancel := context.WithCancel(ctx)

	ns := rawCtx.Namespace
	if ns == "" {
		ns = "default"
	}

	conn := &Connection{
		KubeContext: KubeContext{
			Name:      contextName,
			Cluster:   rawCtx.Cluster,
			User:      rawCtx.AuthInfo,
			Namespace: ns,
			Status:    StatusConnected,
		},
		Config:    restCfg,
		Clientset: clientset,
		Dynamic:   dynClient,
		Discovery: disc,
		cancel:    cancel,
	}

	m.mu.Lock()
	m.connections[contextName] = conn
	m.mu.Unlock()

	m.emitStatus(contextName, StatusConnected)

	go m.healthMonitor(connCtx, conn)
	go m.emitDiscovery(contextName)
	go func() {
		if sv, err := conn.Clientset.Discovery().ServerVersion(); err == nil {
			provider := detectProvider(rawCtx.Cluster, sv.GitVersion)
			m.mu.Lock()
			if c, ok := m.connections[contextName]; ok {
				c.ServerVersion = sv.GitVersion
				c.Provider = provider
			}
			m.mu.Unlock()
			m.emitEvent(fmt.Sprintf("metadata:%s:cluster", contextName), map[string]string{
				"serverVersion": sv.GitVersion,
				"provider":      provider,
			})
		}
	}()

	return nil
}

func (m *Manager) emitDiscovery(contextName string) {
	resources, err := m.DiscoverResources(contextName)
	if err != nil {
		slox.Warn(m.ctx, "resource discovery failed", "context", contextName, "error", err)
		return
	}
	m.emitEvent(fmt.Sprintf("discovery:%s:resources", contextName), resources)
}

func (m *Manager) Disconnect(contextName string) error {
	m.mu.Lock()
	conn, ok := m.connections[contextName]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.connections, contextName)
	m.mu.Unlock()

	conn.cancel()
	m.emitStatus(contextName, StatusDisconnected)
	return nil
}

func (m *Manager) GetConnection(contextName string) (*Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.connections[contextName]
	if !ok {
		return nil, fmt.Errorf("not connected to %q", contextName)
	}
	return conn, nil
}

func (m *Manager) ListNamespaces(ctx context.Context, contextName string) ([]string, error) {
	conn, err := m.GetConnection(contextName)
	if err != nil {
		return nil, err
	}

	nsList, err := conn.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces for %q: %w", contextName, err)
	}

	names := make([]string, len(nsList.Items))
	for i, ns := range nsList.Items {
		names[i] = ns.Name
	}
	return names, nil
}

func (m *Manager) CreateNamespace(ctx context.Context, contextName, name string) error {
	conn, err := m.GetConnection(contextName)
	if err != nil {
		return err
	}
	_, err = conn.Clientset.CoreV1().Namespaces().Create(ctx,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}},
		metav1.CreateOptions{})
	return err
}

func (m *Manager) DeleteNamespace(ctx context.Context, contextName, name string) error {
	conn, err := m.GetConnection(contextName)
	if err != nil {
		return err
	}
	return conn.Clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

type APIResource struct {
	GVR        string `json:"gvr"`
	Kind       string `json:"kind"`
	Namespaced bool   `json:"namespaced"`
}

func (m *Manager) DiscoverResources(contextName string) ([]APIResource, error) {
	conn, err := m.GetConnection(contextName)
	if err != nil {
		return nil, err
	}

	lists, err := conn.Discovery.ServerPreferredResources()
	if err != nil {
		// partial results are OK
		if lists == nil {
			return nil, fmt.Errorf("discovering resources for %q: %w", contextName, err)
		}
	}

	var resources []APIResource
	for _, list := range lists {
		gv := list.GroupVersion
		var group, version string
		if idx := strings.LastIndex(gv, "/"); idx != -1 {
			group = gv[:idx]
			version = gv[idx+1:]
		} else {
			group = ""
			version = gv
		}
		gKey := group
		if gKey == "" {
			gKey = "core"
		}

		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") {
				continue
			}
			resources = append(resources, APIResource{
				GVR:        fmt.Sprintf("%s.%s.%s", gKey, version, r.Name),
				Kind:       r.Kind,
				Namespaced: r.Namespaced,
			})
		}
	}

	return resources, nil
}

func (m *Manager) DisconnectAll() error {
	m.mu.RLock()
	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.Disconnect(name); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) healthMonitor(ctx context.Context, conn *Connection) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			body, err := conn.Clientset.Discovery().RESTClient().Get().AbsPath("/healthz").Do(ctx).Raw()
			if err != nil {
				slox.Warn(m.ctx, "health check failed", "context", conn.Name, "error", err)
				m.mu.Lock()
				if c, ok := m.connections[conn.Name]; ok {
					c.Status = StatusError
				}
				m.mu.Unlock()
				m.emitStatus(conn.Name, StatusError)
			} else if strings.TrimSpace(string(body)) == "ok" {
				m.mu.Lock()
				if c, ok := m.connections[conn.Name]; ok && c.Status != StatusConnected {
					c.Status = StatusConnected
					m.mu.Unlock()
					m.emitStatus(conn.Name, StatusConnected)
				} else {
					m.mu.Unlock()
				}
			}
		}
	}
}

func (m *Manager) emitStatus(contextName string, status ConnectionStatus) {
	if m.emitEvent != nil {
		m.emitEvent(fmt.Sprintf("status:%s:connection", contextName), status.String())
	}
	slox.Info(m.ctx, "cluster status changed", "context", contextName, "status", status)
}
