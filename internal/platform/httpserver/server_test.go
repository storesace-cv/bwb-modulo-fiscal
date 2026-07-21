package httpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/health"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/httpserver"
)

func TestHTTPHealthIntegration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	addr := ln.Addr().String()

	mux := http.NewServeMux()
	mux.Handle("/v1/health", health.NewHandler("int-1.0.0", "AO-INT"))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httpserver.New(httpserver.Config{
		Addr:              addr,
		ReadTimeout:       2 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       2 * time.Second,
		ShutdownTimeout:   2 * time.Second,
	}, mux, logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	waitReady(t, "http://"+addr+"/v1/health")

	t.Run("valid_get", func(t *testing.T) {
		resp, err := http.Get("http://" + addr + "/v1/health")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q", ct)
		}
		var body health.Response
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Status != health.StatusOK || body.Version != "int-1.0.0" || body.FiscalPackage != "AO-INT" {
			t.Fatalf("unexpected body: %+v", body)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		resp, err := http.Post("http://"+addr+"/v1/health", "application/json", nil)
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", resp.StatusCode)
		}
	})

	t.Run("graceful_shutdown", func(t *testing.T) {
		ctx := context.Background()
		if err := srv.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Serve after shutdown: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for Serve to return")
		}

		_, err := http.Get("http://" + addr + "/v1/health")
		if err == nil {
			t.Fatal("expected connection error after shutdown")
		}
	})
}

func TestNewSetsMaxHeaderBytes(t *testing.T) {
	if httpserver.MaxHeaderBytes != 64<<10 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", httpserver.MaxHeaderBytes, 64<<10)
	}
}

func waitReady(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server not ready at %s", url)
}
