package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go-boilerplate/internal/golibs/configs"

	"github.com/spf13/cobra"
)

// newHealthcheckCmd probes the local monitor server. Distroless images have no
// curl, so Docker / ECS container health checks run this instead.
func newHealthcheckCmd(configDir *string) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "healthcheck <service>",
		Short: "Exit 0 if the service's /livez (or --path) answers 200",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configs.Load[configs.Base](args[0], *configDir)
			if err != nil {
				return err
			}
			if cfg.Common.MetricsPort == 0 {
				return fmt.Errorf("%s has no metrics_port", args[0])
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Second)
			defer cancel()
			url := fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Common.MetricsPort, path)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("%s returned %s", path, resp.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", "/livez", "probe path: /livez or /readyz")
	return cmd
}
