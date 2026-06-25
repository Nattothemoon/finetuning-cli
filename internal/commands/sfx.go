package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

// newSfxCmd wires `ft sfx` — sound-effect generation. SFX has its own endpoints
// (/v1/sound-effects) and store, separate from music/instrumental, so it gets
// its own command group rather than reusing `ft list`/`ft get`.
func newSfxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sfx",
		Short: "Generate and manage sound effects",
		Long:  "Create short AI sound effects and list, inspect, or download them. Sound effects draw from a shared daily cap (100/UTC day) across the API and web app.",
	}
	cmd.AddCommand(
		newSfxGenerateCmd(),
		newSfxListCmd(),
		newSfxGetCmd(),
		newSfxDownloadCmd(),
	)
	return cmd
}

// sfxPollConfig is tuned tighter than music: sound effects render in seconds.
func sfxPollConfig() pollConfig {
	return pollConfig{
		Initial: 1500 * time.Millisecond,
		Max:     5 * time.Second,
		Factor:  1.5,
		Timeout: 2 * time.Minute,
	}
}

func newSfxGenerateCmd() *cobra.Command {
	var (
		duration   float64
		enhance    bool
		outputPath string
		noWait     bool
		jsonOut    bool
		webhook    string
	)
	cmd := &cobra.Command{
		Use:     "generate <prompt>",
		Aliases: []string{"gen"},
		Short:   "Create a sound effect; by default polls until done, then downloads the MP3",
		Long: "Generate a short sound effect from a prose prompt. " +
			"Note: if you cancel mid-poll the effect still renders server-side and still counts toward your daily cap; " +
			"resume with `ft sfx get <id>` or `ft sfx download <id>`.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			prompt := strings.Join(args, " ")
			req := &api.CreateSoundEffectRequest{
				Prompt:   prompt,
				Duration: duration,
				Enhance:  enhance,
				Webhook:  webhook,
			}
			s, err := c.CreateSoundEffect(cmd.Context(), req)
			if err != nil {
				return err
			}

			if noWait || webhook != "" {
				if jsonOut {
					return output.RawJSON(os.Stdout, s)
				}
				fmt.Fprintf(os.Stderr, "Queued. id=%s status=%s\n", s.ID, s.Status)
				return nil
			}
			finished, err := waitForSoundEffect(cmd.Context(), c, s.ID, sfxPollConfig())
			if err != nil {
				return err
			}
			if finished.Status == api.StatusFailed {
				return fmt.Errorf("sound effect failed: %s", finished.ErrorMessage)
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, finished)
			}
			path, err := downloadSfxTo(cmd.Context(), c, finished, outputPath)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Saved to %s\n", path)
			return nil
		},
	}
	f := cmd.Flags()
	f.Float64Var(&duration, "duration", 4, "length in seconds (1-8, snapped to 0.5s steps)")
	f.BoolVar(&enhance, "enhance", true, "let the model expand your prompt; pass --enhance=false to use it verbatim")
	f.StringVarP(&outputPath, "output", "o", "", "MP3 destination (default <slug>-<short-id>.mp3 in cwd)")
	f.BoolVar(&noWait, "no-wait", false, "return after the 202 instead of polling")
	f.BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout (instead of downloading)")
	f.StringVar(&webhook, "webhook", "", "https callback URL; implies --no-wait")
	return cmd
}

func newSfxListCmd() *cobra.Command {
	var (
		limit   int
		offset  int
		status  string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent sound effects",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			if limit < 1 || limit > 99 {
				return fmt.Errorf("--limit must be between 1 and 99 (got %d)", limit)
			}
			opts := api.ListOptions{Limit: limit, Offset: offset}
			if status != "" {
				opts.Status = api.GenerationStatus(status)
			}
			res, err := c.ListSoundEffects(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, res)
			}
			if len(res.SoundEffects) == 0 {
				fmt.Fprintln(os.Stderr, "No sound effects yet — try `ft sfx generate \"thunderclap with heavy rain\"`.")
				return nil
			}
			output.SoundEffectTable(os.Stdout, res.SoundEffects)
			if res.HasMore {
				fmt.Fprintf(os.Stderr, "\n…more available. Re-run with --offset %d.\n", res.NextOffset)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.IntVar(&limit, "limit", 20, "rows per page (1-99)")
	f.IntVar(&offset, "offset", 0, "pagination offset")
	f.StringVar(&status, "status", "", "filter by status: pending|processing|completed|failed")
	f.BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout")
	return cmd
}

func newSfxGetCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one sound effect's detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			s, err := c.GetSoundEffect(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, s)
			}
			output.SoundEffectDetail(os.Stdout, s)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout")
	return cmd
}

func newSfxDownloadCmd() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "download <id>",
		Short: "Download a completed sound effect's MP3",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			s, err := c.GetSoundEffect(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if s.Status != api.StatusCompleted {
				return fmt.Errorf("sound effect is %s — only completed effects can be downloaded", s.Status)
			}
			path, err := downloadSfxTo(cmd.Context(), c, s, outputPath)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Saved to %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "destination file path (default: <slug>-<short-id>.mp3 in cwd)")
	return cmd
}

// waitForSoundEffect polls GET /v1/sound-effects/:id until terminal or timeout.
// Mirrors waitForCompletion (music) but on the SFX endpoint and a tighter cadence.
func waitForSoundEffect(ctx context.Context, c *api.Client, id string, pcfg pollConfig) (*api.SoundEffect, error) {
	deadline := time.Now().Add(pcfg.Timeout)
	interval := pcfg.Initial
	start := time.Now()

	var sp *spinner.Spinner
	if output.StderrIsTTY() {
		sp = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
		sp.Suffix = " Generating..."
		sp.Start()
		defer sp.Stop()
	}

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("still processing after %s — run `ft sfx get %s` later", pcfg.Timeout, id)
		}
		s, err := c.GetSoundEffect(ctx, id)
		if err != nil {
			if apiErr, ok := api.IsAPIError(err); ok && apiErr.HTTPStatus == 429 {
				wait := time.Duration(apiErr.RetryAfter) * time.Second
				if wait < time.Second {
					wait = 5 * time.Second
				}
				if err := sleep(ctx, wait); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}
		if sp != nil {
			sp.Suffix = fmt.Sprintf(" Generating... (%ds elapsed, status=%s)", int(time.Since(start).Seconds()), s.Status)
		}
		switch s.Status {
		case api.StatusCompleted, api.StatusFailed:
			if sp != nil {
				sp.FinalMSG = fmt.Sprintf("✓ %s in %ds\n", s.Status, int(time.Since(start).Seconds()))
			}
			return s, nil
		}
		if err := sleep(ctx, interval); err != nil {
			return nil, err
		}
		interval = time.Duration(float64(interval) * pcfg.Factor)
		if interval > pcfg.Max {
			interval = pcfg.Max
		}
	}
}

// downloadSfxTo streams the effect's clip to disk and returns the path written.
// Mirrors downloadTo (music) but keys off SoundEffect fields.
func downloadSfxTo(ctx context.Context, c *api.Client, s *api.SoundEffect, outputPath string) (string, error) {
	dest := outputPath
	if dest == "" {
		dest = defaultSfxFilename(s)
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(abs)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", abs, err)
	}
	defer f.Close()

	var lastReport time.Time
	progress := func(written, total int64) {
		if !output.StderrIsTTY() {
			return
		}
		if time.Since(lastReport) < 500*time.Millisecond {
			return
		}
		lastReport = time.Now()
		if total > 0 {
			pct := float64(written) * 100 / float64(total)
			fmt.Fprintf(os.Stderr, "\rDownloading %s / %s (%.0f%%)", output.HumanBytes(written), output.HumanBytes(total), pct)
		} else {
			fmt.Fprintf(os.Stderr, "\rDownloading %s", output.HumanBytes(written))
		}
	}
	_, err = c.DownloadAudio(ctx, s.AudioURL, f, progress)
	if output.StderrIsTTY() {
		fmt.Fprintln(os.Stderr)
	}
	if err != nil {
		_ = os.Remove(abs)
		return "", err
	}
	return abs, nil
}

func defaultSfxFilename(s *api.SoundEffect) string {
	slug := slugify(s.Prompt)
	if slug == "" {
		slug = "sound-effect"
	}
	return fmt.Sprintf("%s-%s.mp3", slug, output.ShortID(s.ID))
}
