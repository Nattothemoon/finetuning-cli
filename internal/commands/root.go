// Package commands wires cobra subcommands and shared global flags.
package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/auth"
	"github.com/finetuning/cli/internal/config"
	"github.com/spf13/cobra"
)

// GlobalFlags are the values exposed at the root level. Resolved once and
// stashed on the cobra context so child commands don't re-parse.
type GlobalFlags struct {
	APIKey  string
	BaseURL string
	Config  string
	Verbose bool
	NoColor bool
}

type ctxKey struct{ name string }

var (
	flagsKey  = ctxKey{"flags"}
	clientKey = ctxKey{"client"}
	configKey = ctxKey{"config"}
)

// withFlags stashes resolved flags + a configured api.Client into the context.
// Commands that need either retrieve them via getClient / getFlags.
func withFlags(ctx context.Context, gf *GlobalFlags, cfg config.Config, client *api.Client) context.Context {
	ctx = context.WithValue(ctx, flagsKey, gf)
	ctx = context.WithValue(ctx, configKey, cfg)
	ctx = context.WithValue(ctx, clientKey, client)
	return ctx
}

func getFlags(cmd *cobra.Command) *GlobalFlags {
	v, _ := cmd.Context().Value(flagsKey).(*GlobalFlags)
	return v
}

func getClient(cmd *cobra.Command) *api.Client {
	v, _ := cmd.Context().Value(clientKey).(*api.Client)
	return v
}

func getConfig(cmd *cobra.Command) config.Config {
	if v, ok := cmd.Context().Value(configKey).(config.Config); ok {
		return v
	}
	return config.Default()
}

// NewRootCmd is the entry point used by main.
func NewRootCmd() *cobra.Command {
	gf := &GlobalFlags{}

	cmd := &cobra.Command{
		Use:           "ft",
		Short:         "Finetuning AI music generation from the command line",
		Long:          "ft is the command-line companion to finetuning.ai — generate, list, and download AI music from your terminal.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       api.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Resolve config + key + base URL once for all subcommands.
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			baseURL := cfg.BaseURL
			if gf.BaseURL != "" {
				baseURL = gf.BaseURL
			}
			// Auth subcommands tolerate a missing key — they set it.
			key, _, _ := auth.Get(gf.APIKey)

			client := api.NewClient(baseURL, key)
			if gf.Verbose {
				client.VerboseLog = func(format string, args ...any) {
					fmt.Fprintf(os.Stderr, "[ft] "+format+"\n", args...)
				}
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			cmd.SetContext(withFlags(ctx, gf, cfg, client))
			_ = cancel // signal.NotifyContext returns cancel for explicit teardown; cobra owns the lifetime here.
			return nil
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&gf.APIKey, "api-key", "", "API key override (defaults to FINETUNING_API_KEY env or keychain)")
	flags.StringVar(&gf.BaseURL, "base-url", "", "override the API base URL (testing only)")
	flags.StringVar(&gf.Config, "config", "", "custom config file path (not yet used)")
	flags.BoolVarP(&gf.Verbose, "verbose", "v", false, "log HTTP requests to stderr")
	flags.BoolVar(&gf.NoColor, "no-color", false, "disable ANSI colors")

	cmd.AddCommand(
		newAuthCmd(),
		newMeCmd(),
		newGenerateCmd(),
		newListCmd(),
		newGetCmd(),
		newDownloadCmd(),
		newDeleteCmd(),
		newPlaylistsCmd(),
		newPlaylistCmd(),
		newDoctorCmd(),
		newUpdateCmd(),
	)
	return cmd
}

// requireKey is called by commands that need authentication. It returns a
// user-facing error if no key is configured.
func requireKey(c *api.Client) error {
	if c.APIKey == "" {
		return fmt.Errorf("no API key configured — run `ft auth login`")
	}
	return nil
}
