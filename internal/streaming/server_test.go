package streaming

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestServerStartsOnRandomPort(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)

	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	testza.AssertTrue(t, srv.Port() > 0)
}

func TestTokenIs64CharHex(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)

	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	testza.AssertEqual(t, 64, len(srv.Token()))
}

func TestNoTokenReturns401(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)

	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health/bad-token", srv.Port()))
	testza.AssertNoError(t, err)
	defer resp.Body.Close()
	testza.AssertEqual(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestValidTokenReturns200(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)

	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health/%s", srv.Port(), srv.Token()))
	testza.AssertNoError(t, err)
	defer resp.Body.Close()
	testza.AssertEqual(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	testza.AssertEqual(t, "ok", string(body))
}

func TestStopShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)

	testza.AssertNoError(t, srv.Start(ctx))

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health/%s", srv.Port(), srv.Token()))
	testza.AssertNoError(t, err)
	resp.Body.Close()
	testza.AssertEqual(t, http.StatusOK, resp.StatusCode)

	testza.AssertNoError(t, srv.Stop())
}

func TestWSLogsRouteRejectsNoToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)
	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/bad-token/ws/logs/some-stream", srv.Port()))
	testza.AssertNoError(t, err)
	defer resp.Body.Close()
	testza.AssertEqual(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWSExecRouteRejectsNoToken(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)
	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/bad-token/ws/exec/some-session", srv.Port()))
	testza.AssertNoError(t, err)
	defer resp.Body.Close()
	testza.AssertEqual(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRegisterHandlers_NilSafe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(string, any) {}, ctx)
	// RegisterHandlers with nil values should not panic
	srv.RegisterHandlers(nil, nil)
	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()
}

func TestEmitsStreamingReady(t *testing.T) {
	var emitted bool
	var cfg StreamingConfig

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := NewServer(func(name string, data any) {
		if name == "streaming:ready" {
			emitted = true
			cfg = data.(StreamingConfig)
		}
	}, ctx)

	testza.AssertNoError(t, srv.Start(ctx))
	defer srv.Stop()

	testza.AssertTrue(t, emitted)
	testza.AssertTrue(t, cfg.Port > 0)
	testza.AssertEqual(t, 64, len(cfg.Token))
}
