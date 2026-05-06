package cli

import (
	"log/slog"
	"os"

	"github.com/myorg/myapp/internal/version"
	"github.com/spf13/cobra"
)

var (
	logLevel  string
	logFormat string
)

// NewRootCmd creates the root cobra command with all subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "myapp",
		Short: "One-line description of myapp",
		Long:  `Longer description. Replace with the actual purpose of this binary.`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			setupLogger()
		},
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	root.PersistentFlags().StringVar(&logFormat, "log-format", "json", "log format (json, text)")

	root.AddCommand(
		newVersionCmd(),
		newServeCmd(),
	)

	return root
}

func setupLogger() {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if logFormat == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler).With("commit", version.CommitShort()))
}
