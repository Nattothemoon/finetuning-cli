package commands

import (
	"fmt"
	"os"
	"runtime"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/auth"
	"github.com/finetuning/cli/internal/config"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check health of the API, config, and stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			cfgPath, _ := config.Path()
			fmt.Fprintf(os.Stdout, "ft version:    %s (%s/%s, go %s)\n", api.Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
			fmt.Fprintf(os.Stdout, "base URL:      %s\n", cfg.BaseURL)
			fmt.Fprintf(os.Stdout, "config file:   %s\n", cfgPath)

			key, source, err := auth.Get(getFlags(cmd).APIKey)
			if err != nil {
				fmt.Fprintf(os.Stdout, "API key:       (none — run `ft auth login`)\n")
			} else {
				fmt.Fprintf(os.Stdout, "API key:       %s (source: %s)\n", redact(key), source)
			}

			c := getClient(cmd)
			h, err := c.Health(cmd.Context())
			if err != nil {
				fmt.Fprintf(os.Stdout, "API health:    ✗ %v\n", err)
				return err
			}
			fmt.Fprintf(os.Stdout, "API health:    ✓ %v\n", h)

			if key != "" {
				me, err := c.Me(cmd.Context())
				if err != nil {
					fmt.Fprintf(os.Stdout, "auth check:    ✗ %v\n", err)
					return err
				}
				fmt.Fprintf(os.Stdout, "auth check:    ✓ %s (%s tier, %d / %d monthly remaining)\n",
					me.Email, me.Tier, me.Limits.MonthlyRemaining, me.Limits.MonthlyGenerations)
			}
			return nil
		},
	}
}

func redact(k string) string {
	if len(k) <= 12 {
		return "****"
	}
	return k[:12] + "****"
}
