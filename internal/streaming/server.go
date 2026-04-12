package streaming

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"path/filepath"

	"github.com/Vilsol/slox"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"

	"github.com/Vilsol/klados/internal/exec"
	"github.com/Vilsol/klados/internal/logs"
)

type StreamingConfig struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

type Server struct {
	app         *fiber.App
	port        int
	token       string
	emitEvent   func(string, any)
	ctx         context.Context
	logStreamer  *logs.Streamer
	execManager *exec.Manager
	pluginsDir  string
}

func (s *Server) SetPluginsDir(dir string) {
	s.pluginsDir = dir
}

func NewServer(emitEvent func(string, any), ctx context.Context) *Server {
	return &Server{
		emitEvent: emitEvent,
		ctx:       ctx,
	}
}

func (s *Server) RegisterHandlers(l *logs.Streamer, e *exec.Manager) {
	s.logStreamer = l
	s.execManager = e
}

func (s *Server) Start(ctx context.Context) error {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generating token: %w", err)
	}
	s.token = hex.EncodeToString(tokenBytes)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listening: %w", err)
	}
	s.port = ln.Addr().(*net.TCPAddr).Port

	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	s.app.Get("/health/:token", func(c *fiber.Ctx) error {
		if c.Params("token") != s.token {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.SendString("ok")
	})

	s.app.Use("/:token", func(c *fiber.Ctx) error {
		if c.Params("token") != s.token {
			slox.Warn(s.ctx, "streaming auth failure", "path", c.Path())
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Next()
	})

	s.app.Get("/:token/ws/logs/:streamID", fiberws.New(func(c *fiberws.Conn) {
		if s.logStreamer != nil {
			s.logStreamer.HandleConn(c.Params("streamID"), c)
		}
	}))

	s.app.Get("/:token/ws/exec/:sessionID", fiberws.New(func(c *fiberws.Conn) {
		if s.execManager != nil {
			s.execManager.HandleConn(c.Params("sessionID"), c)
		}
	}))

	s.app.Post("/:token/log", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		slox.Info(s.ctx, "frontend", "msg", string(c.Body()))
		return c.SendStatus(fiber.StatusNoContent)
	})

	s.app.Options("/:token/log", func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type")
		return c.SendStatus(fiber.StatusNoContent)
	})

	s.app.Get("/:token/plugins/*", func(c *fiber.Ctx) error {
		if c.Params("token") != s.token {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if s.pluginsDir == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		// Dynamic import() from the Wails webview origin requires CORS.
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET")
		filePath := filepath.Join(s.pluginsDir, c.Params("*"))
		switch filepath.Ext(filePath) {
		case ".js", ".mjs":
			c.Set("Content-Type", "application/javascript; charset=utf-8")
		case ".map":
			c.Set("Content-Type", "application/json; charset=utf-8")
		}
		return c.SendFile(filePath)
	})

	if s.emitEvent != nil {
		s.emitEvent("streaming:ready", StreamingConfig{
			Port:  s.port,
			Token: s.token,
		})
	}

	slox.Info(s.ctx, "streaming server started", "port", s.port)

	go func() {
		if err := s.app.Listener(ln); err != nil {
			slox.Error(s.ctx, "streaming server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		_ = s.Stop()
	}()

	return nil
}

func (s *Server) Stop() error {
	if s.app != nil {
		slox.Info(s.ctx, "streaming server stopped")
		return s.app.Shutdown()
	}
	return nil
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) Token() string {
	return s.token
}
