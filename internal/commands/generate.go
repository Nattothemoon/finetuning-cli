package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

// pollConfig controls the poll loop in `ft generate`. Exposed for tests.
type pollConfig struct {
	Initial time.Duration
	Max     time.Duration
	Factor  float64
	Timeout time.Duration
}

func defaultPollConfig() pollConfig {
	return pollConfig{
		Initial: 3 * time.Second,
		Max:     15 * time.Second,
		Factor:  1.3,
		Timeout: 5 * time.Minute,
	}
}

func newGenerateCmd() *cobra.Command {
	var (
		duration   int
		bpm        int
		key        string
		scale      string
		timesig    string
		language   string
		lyrics     string
		seed       int64
		outputPath string
		noWait     bool
		jsonOut    bool
		webhook    string
	)
	cmd := &cobra.Command{
		Use:   "generate <tags>",
		Short: "Create a new generation; by default polls until done, then downloads the MP3",
		Long: "Generate a track from a prose prompt (tags). " +
			"Defaults match the web app. " +
			"Note: if you cancel mid-poll the generation still runs server-side and still costs a credit; " +
			"resume with `ft get <id>` or `ft download <id>`.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			tags := strings.Join(args, " ")
			req := &api.CreateGenerationRequest{
				Tags:          tags,
				Lyrics:        lyrics,
				Duration:      duration,
				BPM:           bpm,
				Key:           key,
				Scale:         scale,
				TimeSignature: timesig,
				Language:      language,
				Seed:          seed,
				Webhook:       webhook,
			}
			g, err := c.CreateGeneration(cmd.Context(), req)
			if err != nil {
				return err
			}
			// Warn on server-side clamping.
			warnClamp(g, duration, bpm)

			if noWait || webhook != "" {
				if jsonOut {
					return output.RawJSON(os.Stdout, g)
				}
				fmt.Fprintf(os.Stderr, "Queued. id=%s status=%s\n", g.ID, g.Status)
				return nil
			}
			finished, err := waitForCompletion(cmd.Context(), c, g.ID, defaultPollConfig())
			if err != nil {
				return err
			}
			if finished.Status == api.StatusFailed {
				return fmt.Errorf("generation failed: %s", finished.ErrorMessage)
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, finished)
			}
			path, err := downloadTo(cmd.Context(), c, finished, outputPath)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Saved to %s\n", path)
			return nil
		},
	}
	f := cmd.Flags()
	f.IntVar(&duration, "duration", 60, "track length in seconds (5-210 for paid tiers)")
	f.IntVar(&bpm, "bpm", 120, "tempo (60-200)")
	f.StringVar(&key, "key", "C", "musical key: C, C#, D, D#, E, F, F#, G, G#, A, A#, B")
	f.StringVar(&scale, "scale", "major", "scale: major|minor")
	f.StringVar(&timesig, "time-sig", "4", "time signature: 2|3|4|5|6|7")
	f.StringVar(&language, "language", "en", "lyric language: en, ja, de, fr, es, zh, ko, pt, it, ru")
	f.StringVar(&lyrics, "lyrics", "", "lyrics text (0-2000 chars)")
	f.Int64Var(&seed, "seed", 0, "reproducibility seed (0 = random server-side)")
	f.StringVarP(&outputPath, "output", "o", "", "MP3 destination (default <slug>-<short-id>.mp3 in cwd)")
	f.BoolVar(&noWait, "no-wait", false, "return after the 202 instead of polling")
	f.BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout (instead of downloading)")
	f.StringVar(&webhook, "webhook", "", "https callback URL; implies --no-wait")
	return cmd
}

// waitForCompletion polls /v1/generations/:id until status is terminal or pcfg.Timeout elapses.
// Honors 429 Retry-After via api errors; otherwise grows the interval geometrically.
func waitForCompletion(ctx context.Context, c *api.Client, id string, pcfg pollConfig) (*api.Generation, error) {
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
			return nil, fmt.Errorf("still processing after %s — run `ft get %s` later", pcfg.Timeout, id)
		}
		g, err := c.GetGeneration(ctx, id)
		if err != nil {
			// Honor 429 from polling; otherwise propagate.
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
			sp.Suffix = fmt.Sprintf(" Generating... (%ds elapsed, status=%s)", int(time.Since(start).Seconds()), g.Status)
		}
		switch g.Status {
		case api.StatusCompleted, api.StatusFailed:
			if sp != nil {
				sp.FinalMSG = fmt.Sprintf("✓ %s in %ds\n", g.Status, int(time.Since(start).Seconds()))
			}
			return g, nil
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

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return errors.New("interrupted; the generation is still running server-side — resume with `ft get <id>`")
	case <-t.C:
		return nil
	}
}

// warnClamp prints a stderr note when the server clamped the requested values.
func warnClamp(g *api.Generation, reqDuration, reqBPM int) {
	if g == nil || g.Parameters == nil {
		return
	}
	if reqDuration > 0 {
		if got, ok := numericParam(g.Parameters, "duration"); ok && got != reqDuration {
			fmt.Fprintf(os.Stderr, "note: duration clamped from %d → %d\n", reqDuration, got)
		}
	}
	if reqBPM > 0 {
		if got, ok := numericParam(g.Parameters, "bpm"); ok && got != reqBPM {
			fmt.Fprintf(os.Stderr, "note: bpm clamped from %d → %d\n", reqBPM, got)
		}
	}
}

func numericParam(p map[string]any, k string) (int, bool) {
	switch v := p[k].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	}
	return 0, false
}
