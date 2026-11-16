// Command server is the single binary for every service:
//
//	server gserver <service> [--config-dir configs]
//	server migrate <service> up|down [--steps N]
//
// One image, one build; the deployment chooses the service by argument.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "go-boilerplate/cmd/server/auth"
	_ "go-boilerplate/cmd/server/file"
	_ "go-boilerplate/cmd/server/gateway"
	"go-boilerplate/internal/golibs/bootstrap"

	"github.com/spf13/cobra"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var configDir string

	root := &cobra.Command{
		Use:           "server",
		Short:         "Runs any service of this repository",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&configDir, "config-dir", "configs",
		"directory with common.yaml, <service>.yaml and optional secrets/<service>.yaml")

	root.AddCommand(
		&cobra.Command{
			Use:       "gserver <service>",
			Short:     "Start a service",
			Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
			ValidArgs: bootstrap.Names(),
			RunE: func(cmd *cobra.Command, args []string) error {
				return bootstrap.Run(cmd.Context(), args[0], configDir)
			},
		},
		newMigrateCmd(&configDir),
	)
	return root
}
