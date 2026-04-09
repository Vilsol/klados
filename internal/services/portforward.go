package services

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Vilsol/klados/internal/config"
	"github.com/Vilsol/klados/internal/portforward"
)

type PortForwardService struct {
	appService *AppService
	manager    *portforward.Manager
	ctx        context.Context
}

func NewPortForwardService(appSvc *AppService) *PortForwardService {
	return &PortForwardService{appService: appSvc}
}

func (s *PortForwardService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	s.manager = s.appService.PortForwardManager()
	return nil
}

func (s *PortForwardService) ServiceShutdown() error {
	if s.manager != nil {
		s.manager.StopAll()
	}
	return nil
}

func (s *PortForwardService) StartForward(contextName, namespace string, targetKind portforward.TargetKind, targetName, targetGVR string, localPort, remotePort int) (portforward.ForwardSpec, error) {
	return s.manager.StartForward(portforward.ForwardSpec{
		ContextName: contextName,
		Namespace:   namespace,
		TargetKind:  targetKind,
		TargetName:  targetName,
		TargetGVR:   targetGVR,
		LocalPort:   localPort,
		RemotePort:  remotePort,
	})
}

func (s *PortForwardService) StopForward(forwardID string) error {
	return s.manager.StopForward(forwardID)
}

func (s *PortForwardService) ListForwards(contextName string) []portforward.ForwardSpec {
	return s.manager.ListForwards(contextName)
}

func (s *PortForwardService) SavePortForward(ctxName string, fwd config.SavedPortForward) error {
	return s.manager.SaveForward(ctxName, fwd)
}

func (s *PortForwardService) RemoveSavedPortForward(ctxName, id string) error {
	return s.manager.RemoveSavedForward(ctxName, id)
}

func (s *PortForwardService) SetPortForwardEnabled(ctxName, id string, enabled bool) error {
	return s.manager.SetForwardEnabled(ctxName, id, enabled)
}

func (s *PortForwardService) ListSavedPortForwards(ctxName string) []config.SavedPortForward {
	return s.manager.ListSavedForwards(ctxName)
}
