package services

import (
	"context"
	"log/slog"

	"github.com/Vilsol/slox"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Vilsol/klados/internal/cluster"
	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/klados/internal/exec"
	"github.com/Vilsol/klados/internal/logs"
	"github.com/Vilsol/klados/internal/portforward"
	"github.com/Vilsol/klados/internal/session"
	"github.com/Vilsol/klados/internal/streaming"
)

type AppService struct {
	clusterMgr         *cluster.Manager
	streamingSrv       *streaming.Server
	logStreamer         *logs.Streamer
	execManager        *exec.Manager
	portForwardManager *portforward.Manager
	session            *session.Session
	config             *config.Config
	pluginSvc          *PluginService
	ctx                context.Context
	app                *application.App
}

func NewAppService(cfg *config.Config, sess *session.Session, ctx context.Context) *AppService {
	return &AppService{
		config:  cfg,
		session: sess,
		ctx:     ctx,
	}
}

func (a *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.app = application.Get()
	a.ctx = slox.Into(ctx, slog.Default())

	emitEvent := func(name string, data any) {
		if a.app != nil {
			a.app.Event.Emit(name, data)
		}
	}

	a.clusterMgr = cluster.NewManager(emitEvent, a.config, a.ctx)
	a.logStreamer = logs.NewStreamer(a.clusterMgr, a.ctx)
	a.execManager = exec.NewManager(a.clusterMgr, a.ctx)
	a.portForwardManager = portforward.NewManager(a.clusterMgr, emitEvent, a.ctx)
	a.streamingSrv = streaming.NewServer(emitEvent, a.ctx)
	a.streamingSrv.RegisterHandlers(a.logStreamer, a.execManager)

	if err := a.streamingSrv.Start(a.ctx); err != nil {
		return err
	}

	if err := a.clusterMgr.LoadKubeconfigs(a.config.KubeconfigPaths); err != nil {
		slox.Warn(a.ctx, "failed to load kubeconfigs", "error", err)
	}

	for _, ctxName := range a.session.ConnectedClusters {
		go func(name string) {
			if err := a.clusterMgr.Connect(a.ctx, name); err != nil {
				slox.Warn(a.ctx, "failed to reconnect cluster", "context", name, "error", err)
			}
		}(ctxName)
	}

	return nil
}

func (a *AppService) ServiceShutdown() error {
	if a.session != nil {
		_ = a.session.Save()
	}

	if a.clusterMgr != nil {
		if err := a.clusterMgr.DisconnectAll(); err != nil {
			slox.Error(a.ctx, "error disconnecting clusters", "error", err)
		}
	}

	if a.streamingSrv != nil {
		if err := a.streamingSrv.Stop(); err != nil {
			slox.Error(a.ctx, "error stopping streaming server", "error", err)
		}
	}

	return nil
}

func (a *AppService) ClusterManager() *cluster.Manager {
	return a.clusterMgr
}

func (a *AppService) Config() *config.Config {
	return a.config
}

func (a *AppService) BrowseKubeconfigFile() (string, error) {
	return a.app.Dialog.OpenFile().
		AddFilter("Kubeconfig files", "*.yaml").
		PromptForSingleSelection()
}

func (a *AppService) BrowsePluginFile() (string, error) {
	return a.app.Dialog.OpenFile().
		AddFilter("Klados plugin archives", "*.oci.tar.gz").
		PromptForSingleSelection()
}

func (a *AppService) GetStreamingConfig() streaming.StreamingConfig {
	return streaming.StreamingConfig{
		Port:  a.streamingSrv.Port(),
		Token: a.streamingSrv.Token(),
	}
}

func (a *AppService) LogStreamer() *logs.Streamer {
	return a.logStreamer
}

func (a *AppService) ExecManager() *exec.Manager {
	return a.execManager
}

func (a *AppService) PortForwardManager() *portforward.Manager {
	return a.portForwardManager
}

func (a *AppService) RegisterPluginsDir(dir string) {
	if a.streamingSrv != nil {
		a.streamingSrv.SetPluginsDir(dir)
	}
}

func (a *AppService) Ctx() context.Context {
	return a.ctx
}

func (a *AppService) SetPluginService(svc *PluginService) {
	a.pluginSvc = svc
}

func (a *AppService) PluginService() *PluginService {
	return a.pluginSvc
}

func (a *AppService) GetSession() *session.Session {
	return a.session
}

func (a *AppService) SaveUIState(openTabs []session.TabState, activeTab int, sidebarCollapsed bool) {
	a.session.OpenTabs = openTabs
	a.session.ActiveTab = activeTab
	a.session.SidebarCollapsed = sidebarCollapsed
	a.session.SaveDebounced()
}

func (a *AppService) LogFrontend(level, message, detail string) {
	args := []any{"source", "frontend"}
	if detail != "" {
		args = append(args, "detail", detail)
	}
	switch level {
	case "warn":
		slox.Warn(a.ctx, message, args...)
	case "error":
		slox.Error(a.ctx, message, args...)
	default:
		slox.Info(a.ctx, message, args...)
	}
}
