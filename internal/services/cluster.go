package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/klados/internal/session"
	"github.com/adrg/xdg"
	"github.com/wailsapp/wails/v3/pkg/application"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterService struct {
	appService *AppService
	session    *session.Session
	ctx        context.Context
}

func NewClusterService(appSvc *AppService, sess *session.Session) *ClusterService {
	return &ClusterService{
		appService: appSvc,
		session:    sess,
	}
}

func (c *ClusterService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	c.ctx = ctx
	return nil
}

func (c *ClusterService) manager() *cluster.Manager {
	return c.appService.ClusterManager()
}

func (c *ClusterService) ListContexts() []cluster.KubeContext {
	return c.manager().ListContexts()
}

func (c *ClusterService) Connect(contextName string) error {
	if err := c.manager().Connect(c.ctx, contextName); err != nil {
		return err
	}

	c.session.ConnectedClusters = appendUnique(c.session.ConnectedClusters, contextName)
	c.session.SaveDebounced()

	if ps := c.appService.PluginService(); ps != nil {
		ps.EmitClusterEvent("cluster:connected", clusterEventPayload(contextName))
	}

	go c.appService.PortForwardManager().ReconnectSaved(contextName)

	return nil
}

func (c *ClusterService) Disconnect(contextName string) error {
	if err := c.manager().Disconnect(contextName); err != nil {
		return err
	}

	c.session.ConnectedClusters = removeString(c.session.ConnectedClusters, contextName)
	c.session.SaveDebounced()

	if ps := c.appService.PluginService(); ps != nil {
		ps.EmitClusterEvent("cluster:disconnected", clusterEventPayload(contextName))
	}
	return nil
}

func (c *ClusterService) ListNamespaces(contextName string) ([]string, error) {
	return c.manager().ListNamespaces(c.ctx, contextName)
}

func (c *ClusterService) SwitchNamespace(contextName, namespace string) error {
	if c.session.ActiveNamespaces == nil {
		c.session.ActiveNamespaces = make(map[string]string)
	}
	c.session.ActiveNamespaces[contextName] = namespace
	c.session.SaveDebounced()

	if ps := c.appService.PluginService(); ps != nil {
		payload, _ := json.Marshal(map[string]string{"context": contextName, "namespace": namespace})
		ps.EmitClusterEvent("namespace:changed", payload)
	}
	return nil
}

func (c *ClusterService) GetActiveNamespace(contextName string) string {
	if c.session.ActiveNamespaces != nil {
		if ns, ok := c.session.ActiveNamespaces[contextName]; ok {
			return ns
		}
	}
	return ""
}

func (c *ClusterService) GetStatus(contextName string) cluster.ConnectionStatus {
	conn, err := c.manager().GetConnection(contextName)
	if err != nil {
		return cluster.StatusDisconnected
	}
	return conn.Status
}

func (c *ClusterService) CreateNamespace(contextName, name string) error {
	return c.manager().CreateNamespace(c.ctx, contextName, name)
}

func (c *ClusterService) DeleteNamespace(contextName, name string) error {
	return c.manager().DeleteNamespace(c.ctx, contextName, name)
}

func (c *ClusterService) AddKubeconfigPath(path string) ([]cluster.KubeContext, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}
	cfg := c.appService.Config()
	if err := cfg.Update(func(cfg *config.Config) {
		for _, p := range cfg.KubeconfigPaths {
			if p == path {
				return
			}
		}
		cfg.KubeconfigPaths = append(cfg.KubeconfigPaths, path)
	}); err != nil {
		return nil, err
	}
	if err := c.manager().LoadKubeconfigs(cfg.KubeconfigPaths); err != nil {
		return nil, err
	}
	return c.manager().ListContexts(), nil
}

func (c *ClusterService) ImportKubeconfigContent(content string) ([]cluster.KubeContext, error) {
	if _, err := clientcmd.Load([]byte(content)); err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}
	dir := filepath.Join(xdg.ConfigHome, "klados", "kubeconfigs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(content))
	path := filepath.Join(dir, fmt.Sprintf("%x", sum[:4])+".yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return nil, err
	}
	return c.AddKubeconfigPath(path)
}

func clusterEventPayload(contextName string) []byte {
	b, _ := json.Marshal(map[string]string{"context": contextName})
	return b
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func removeString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
