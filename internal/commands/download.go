package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

func newDownloadCmd() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "download <id>",
		Short: "Download a completed generation's MP3",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			g, err := c.GetGeneration(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if g.Status != api.StatusCompleted {
				return fmt.Errorf("generation is %s — only completed tracks can be downloaded", g.Status)
			}
			path, err := downloadTo(cmd.Context(), c, g, outputPath)
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

// downloadTo streams the audio file to disk and returns the absolute path written.
func downloadTo(ctx context.Context, c *api.Client, g *api.Generation, outputPath string) (string, error) {
	dest := outputPath
	if dest == "" {
		dest = defaultFilename(g)
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
		// Throttle: print at most twice per second when stderr is a TTY.
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
	_, err = c.DownloadAudio(ctx, g.AudioURL, f, progress)
	if output.StderrIsTTY() {
		fmt.Fprintln(os.Stderr)
	}
	if err != nil {
		_ = os.Remove(abs)
		return "", err
	}
	return abs, nil
}

func defaultFilename(g *api.Generation) string {
	slug := slugify(g.Title)
	if slug == "" {
		slug = slugify(g.Prompt)
	}
	if slug == "" {
		slug = "generation"
	}
	return fmt.Sprintf("%s-%s.mp3", slug, output.ShortID(g.ID))
}

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = slugStrip.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
		s = strings.TrimRight(s, "-")
	}
	return s
}
