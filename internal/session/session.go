package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Vilsol/slox"
	"github.com/adrg/xdg"
	"github.com/sasha-s/go-deadlock"
)

type Session struct {
	ConnectedClusters []string          `json:"connectedClusters"`
	ActiveNamespaces  map[string]string `json:"activeNamespaces"`
	OpenTabs          []TabState        `json:"openTabs"`
	ActiveTab         int               `json:"activeTab"`
	SidebarCollapsed  bool              `json:"sidebarCollapsed"`
	TerminalFontSize  int               `json:"terminalFontSize"`
	Window            WindowState       `json:"window"`

	mu       deadlock.Mutex
	path     string
	debounce *time.Timer
}

type TabState struct {
	ClusterContext string  `json:"clusterContext"`
	GVR            string  `json:"gvr"`
	Namespace      string  `json:"namespace"`
	Name           string  `json:"name"`
	ScrollPosition float64 `json:"scrollPosition,omitempty"`
}

type WindowState struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func defaultSession() *Session {
	return &Session{
		ConnectedClusters: []string{},
		ActiveNamespaces:  map[string]string{},
		OpenTabs:          []TabState{},
		Window: WindowState{
			Width:  1280,
			Height: 800,
		},
	}
}

func sessionPath() (string, error) {
	return xdg.StateFile(filepath.Join("klados", "session.json"))
}

func Load() (*Session, error) {
	p, err := sessionPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			slox.Info(context.Background(), "session not found, using defaults", "path", p)
			s := defaultSession()
			s.path = p
			return s, nil
		}
		return nil, err
	}

	s := defaultSession()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	s.path = p
	slox.Debug(context.Background(), "session loaded", "path", p)
	return s, nil
}

func (s *Session) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Session) saveLocked() error {
	if s.path == "" {
		p, err := sessionPath()
		if err != nil {
			return err
		}
		s.path = p
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		slox.Error(context.Background(), "session save failed", "path", s.path, "error", err)
		return err
	}
	return nil
}

func (s *Session) SaveDebounced() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.debounce != nil {
		s.debounce.Stop()
	}

	s.debounce = time.AfterFunc(500*time.Millisecond, func() {
		slox.Debug(context.Background(), "session debounced save triggered")
		_ = s.Save()
	})
}
