package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type WindowService struct {
	ctx context.Context
	app *application.App
	mu  sync.Mutex
}

func NewWindowService() *WindowService {
	return &WindowService{}
}

func (w *WindowService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	w.app = application.Get()
	return nil
}

func (w *WindowService) ServiceShutdown() error {
	return nil
}

// OpenPanelWindow creates a new OS window for a bottom panel tab.
// The panelID is used as a query parameter so the frontend can identify which tab to render.
func (w *WindowService) OpenPanelWindow(panelID, title string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	windowName := fmt.Sprintf("panel-%s", panelID)

	if existing, ok := w.app.Window.GetByName(windowName); ok {
		existing.Focus()
		return nil
	}

	win := w.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   windowName,
		Title:  title,
		Width:  800,
		Height: 500,
		URL:    fmt.Sprintf("/?panel=%s", panelID),
	})

	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		w.app.Event.Emit("panel:closed", panelID)
	})

	win.Show()
	return nil
}

//wails:ignore
func (w *WindowService) ClosePanelWindow(panelID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	windowName := fmt.Sprintf("panel-%s", panelID)
	if win, ok := w.app.Window.GetByName(windowName); ok {
		win.Close()
	}
}
