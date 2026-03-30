package services

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Vilsol/klados/internal/logs"
)

type LogService struct {
	appService *AppService
	streamer   *logs.Streamer
	ctx        context.Context
}

func NewLogService(appSvc *AppService) *LogService {
	return &LogService{appService: appSvc}
}

func (s *LogService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	s.streamer = s.appService.LogStreamer()
	return nil
}

func (s *LogService) ServiceShutdown() error {
	if s.streamer != nil {
		s.streamer.StopAll()
	}
	return nil
}

func (s *LogService) StartLogStream(ctxName, ns, podName string, opts logs.LogOptions) (string, error) {
	return s.streamer.StartStream(ctxName, ns, podName, opts)
}

func (s *LogService) StopLogStream(streamID string) {
	s.streamer.StopStream(streamID)
}
