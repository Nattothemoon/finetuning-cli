package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

// newInstrumentalCmd wires `ft instrumental` — the vocals-free generator backed
// by POST /v1/instrumental. Far fewer knobs than `ft generate`: the model picks
// instrumentation and tempo from the prompt, so there's no bpm/key/scale/
// time-sig/lyrics/language. Created tracks land in the same store as music, so
// reads reuse `ft get`, `ft download`, and `ft list --type instrumental`.
func newInstrumentalCmd() *cobra.Command {
	var (
		duration   int
		enhance    bool
		seed       int64
		outputPath string
		noWait     bool
		jsonOut    bool
		webhook    string
	)
	cmd := &cobra.Command{
		Use:     "instrumental <prompt>",
		Aliases: []string{"inst"},
		Short:   "Create an instrumental (vocals-free) track; polls until done, then downloads the MP3",
		Long: "Generate an instrumental track from a prose prompt. " +
			"Unlike `ft generate`, there's no bpm/key/scale/lyrics — the model picks " +
			"instrumentation and tempo for you. " +
			"Note: if you cancel mid-poll the generation still runs server-side and still costs a credit; " +
			"resume with `ft get <id>` or `ft download <id>`.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			prompt := strings.Join(args, " ")
			req := &api.CreateInstrumentalRequest{
				Prompt:   prompt,
				Duration: duration,
				Enhance:  enhance,
				Seed:     seed,
				Webhook:  webhook,
			}
			g, err := c.CreateInstrumental(cmd.Context(), req)
			if err != nil {
				return err
			}
			// Warn on server-side duration clamping (snap to 15s / tier ceiling).
			warnClamp(g, duration, 0)

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
	f.IntVar(&duration, "duration", 120, "track length in seconds (30-210, snapped to 15s; free tier capped at 120)")
	f.BoolVar(&enhance, "enhance", true, "let the model expand your prompt; pass --enhance=false to use it verbatim")
	f.Int64Var(&seed, "seed", 0, "reproducibility seed (0 = random server-side)")
	f.StringVarP(&outputPath, "output", "o", "", "MP3 destination (default <slug>-<short-id>.mp3 in cwd)")
	f.BoolVar(&noWait, "no-wait", false, "return after the 202 instead of polling")
	f.BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout (instead of downloading)")
	f.StringVar(&webhook, "webhook", "", "https callback URL; implies --no-wait")
	return cmd
}
