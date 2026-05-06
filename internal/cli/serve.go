package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/myorg/myapp/internal/server"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run as a long-lived HTTP server",
		Long: `Starts an HTTP server with /healthz and /readyz probes.

Optional env vars:
  MYAPP_LISTEN_ADDR  Listen address (default: :8080)`,
		RunE: runServe,
	}
}

func runServe(_ *cobra.Command, _ []string) error {
	var opts []server.Option
	if addr := os.Getenv("MYAPP_LISTEN_ADDR"); addr != "" {
		opts = append(opts, server.WithListenAddr(addr))
	}

	srv := server.NewServer(opts...)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	return srv.Start(ctx)
}
