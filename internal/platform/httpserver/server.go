// Package httpserver encapsula o servidor HTTP com timeouts e encerramento gracioso.
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// MaxHeaderBytes é o limite técnico explícito do tamanho dos headers HTTP (64 KiB).
const MaxHeaderBytes = 64 << 10

// Config parâmetros de rede do servidor.
type Config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// Server envolve http.Server com arranque e shutdown controlados.
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

// New cria um servidor HTTP com o handler raiz fornecido.
func New(cfg Config, handler http.Handler, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			MaxHeaderBytes:    MaxHeaderBytes,
			BaseContext: func(_ net.Listener) context.Context {
				return context.Background()
			},
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		logger:          logger,
	}
}

// ListenAndServe inicia o listener. Bloqueia até o servidor terminar.
// Devolve nil se o encerramento for esperado (Shutdown/Close).
func (s *Server) ListenAndServe() error {
	s.logger.Info("http_listen_start", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Serve serve no listener já reservado. Bloqueia até o servidor terminar.
// Devolve nil se o encerramento for esperado (Shutdown/Close).
func (s *Server) Serve(ln net.Listener) error {
	s.logger.Info("http_serve_start", "addr", ln.Addr().String())
	err := s.httpServer.Serve(ln)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown encerra conexões de forma graciosa dentro do timeout configurado.
func (s *Server) Shutdown(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, s.shutdownTimeout)
	defer cancel()
	s.logger.Info("http_shutdown_start", "timeout_ms", s.shutdownTimeout.Milliseconds())
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.logger.Error("http_shutdown_failed", "error", err.Error())
		return err
	}
	s.logger.Info("http_shutdown_complete")
	return nil
}

// Addr devolve o endereço configurado (útil em testes antes do listen).
func (s *Server) Addr() string {
	return s.httpServer.Addr
}
