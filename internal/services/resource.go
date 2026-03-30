package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/resource"
	"github.com/Vilsol/klados/internal/watcher"
)

type ResourceService struct {
	appService  *AppService
	engine      *resource.ResourceEngine
	watchMgr    *watcher.WatchManager
	registry    *resource.Registry
	enricherReg *resource.EnricherRegistry
	ctx         context.Context
}

func NewResourceService(appSvc *AppService) *ResourceService {
	return &ResourceService{appService: appSvc}
}

func (s *ResourceService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx

	app := application.Get()
	emit := func(name string, data any) {
		if app != nil {
			app.Event.Emit(name, data)
		}
	}

	reg, err := resource.NewRegistry()
	if err != nil {
		return err
	}

	enricherReg := resource.NewEnricherRegistry()
	if err := resource.RegisterBuiltin(reg, enricherReg); err != nil {
		return err
	}

	s.registry = reg
	s.enricherReg = enricherReg
	s.engine = resource.NewResourceEngine(s.appService.ClusterManager(), enricherReg)
	s.watchMgr = watcher.NewWatchManager(s.appService.ClusterManager(), enricherReg, emit, s.appService.Ctx())
	return nil
}

func (s *ResourceService) ServiceShutdown() error {
	if s.watchMgr != nil {
		s.watchMgr.StopAll()
	}
	return nil
}

func (s *ResourceService) ListResources(contextName, gvr, namespace string) ([]map[string]any, error) {
	return s.engine.List(s.ctx, contextName, gvr, namespace)
}

func (s *ResourceService) GetResource(contextName, gvr, namespace, name string) (map[string]any, error) {
	return s.engine.Get(s.ctx, contextName, gvr, namespace, name)
}

func (s *ResourceService) DeleteResource(contextName, gvr, namespace, name string) error {
	return s.engine.Delete(s.ctx, contextName, gvr, namespace, name)
}

func (s *ResourceService) StartWatch(contextName, gvr, namespace string) error {
	return s.watchMgr.StartWatch(contextName, gvr, namespace)
}

func (s *ResourceService) StopWatch(contextName, gvr, namespace string) {
	s.watchMgr.StopWatch(contextName, gvr, namespace)
}

func (s *ResourceService) ListAPIResources(contextName string) ([]cluster.APIResource, error) {
	return s.appService.ClusterManager().DiscoverResources(contextName)
}

func (s *ResourceService) GetDescriptors() []*resource.Descriptor {
	return s.registry.List()
}

func (s *ResourceService) Registry() *resource.Registry {
	return s.registry
}

func (s *ResourceService) EnricherRegistry() *resource.EnricherRegistry {
	return s.enricherReg
}

func (s *ResourceService) Engine() *resource.ResourceEngine {
	return s.engine
}

func (s *ResourceService) WatchMgr() *watcher.WatchManager {
	return s.watchMgr
}

func (s *ResourceService) CreateResource(contextName, gvr, namespace string, obj map[string]any) (map[string]any, error) {
	return s.engine.Create(s.ctx, contextName, gvr, namespace, obj)
}

func (s *ResourceService) UpdateResource(contextName, gvr, namespace string, obj map[string]any) (map[string]any, error) {
	return s.engine.Update(s.ctx, contextName, gvr, namespace, obj)
}

func (s *ResourceService) ForceDeleteResource(contextName, gvr, namespace, name string) error {
	return s.engine.ForceDelete(s.ctx, contextName, gvr, namespace, name)
}

func (s *ResourceService) GetEvents(contextName, namespace, uid string) ([]map[string]any, error) {
	conn, err := s.appService.ClusterManager().GetConnection(contextName)
	if err != nil {
		return nil, err
	}

	fieldSelector := fmt.Sprintf("involvedObject.uid=%s", uid)
	list, err := conn.Clientset.CoreV1().Events(namespace).List(s.ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}

	result := make([]map[string]any, len(list.Items))
	for i := range list.Items {
		data, err := json.Marshal(list.Items[i])
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *ResourceService) ScaleResource(contextName, gvr, namespace, name string, replicas int32) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas))
	_, err := s.engine.Patch(s.ctx, contextName, gvr, namespace, name, types.MergePatchType, patch)
	return err
}

func (s *ResourceService) RestartResource(contextName, gvr, namespace, name string) error {
	ts := time.Now().UTC().Format(time.RFC3339)
	patch := []byte(fmt.Sprintf(
		`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		ts,
	))
	_, err := s.engine.Patch(s.ctx, contextName, gvr, namespace, name, types.MergePatchType, patch)
	return err
}
