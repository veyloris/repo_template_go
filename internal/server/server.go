package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Server is a minimal HTTP server with health probes and graceful shutdown.
// Replace the handler wiring in NewServer with your application's routes.
type Server struct {
	listenAddr string
	logger     *slog.Logger

	// ready is set true when the server is ready to serve traffic. Flip it
	// after any startup work that should gate /readyz (warm caches, prime
	// connections, complete an initial sync, etc.).
	ready atomic.Bool
}

// Option configures the server.
type Option func(*Server)

// WithListenAddr sets the listen address (default ":8080").
func WithListenAddr(addr string) Option {
	return func(s *Server) { s.listenAddr = addr }
}

// WithLogger sets a custom logger (default slog.Default with a "server" tag).
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.logger = l }
}

// NewServer builds a Server with the given options.
func NewServer(opts ...Option) *Server {
	s := &Server{
		listenAddr: ":8080",
		logger:     slog.Default().With("component", "server"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start runs the HTTP server until ctx is cancelled. On cancel, Shutdown is
// called with a 30-second timeout drained from a fresh background context so
// the graceful drain is not pre-empted by the cancelled parent.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)

	srv := &http.Server{
		Addr:         s.listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("starting server", "addr", s.listenAddr)
		s.ready.Store(true)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx) //nolint:contextcheck // detached on purpose: parent is already cancelled
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.ready.Load() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("not ready"))
}
