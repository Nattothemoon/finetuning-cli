package commands

import (
	"fmt"
	"os"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		limit    int
		offset   int
		status   string
		typeKind string
		jsonOut  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent generations",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			if limit < 1 || limit > 99 {
				return fmt.Errorf("--limit must be between 1 and 99 (got %d)", limit)
			}
			if typeKind != "" && typeKind != "music" && typeKind != "instrumental" {
				return fmt.Errorf("--type must be music or instrumental (got %q)", typeKind)
			}
			opts := api.ListOptions{Limit: limit, Offset: offset, Type: typeKind}
			if status != "" {
				opts.Status = api.GenerationStatus(status)
			}
			res, err := c.ListGenerations(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, res)
			}
			if len(res.Generations) == 0 {
				fmt.Fprintln(os.Stderr, "No generations yet — try `ft generate \"some prompt\"`.")
				return nil
			}
			output.GenerationTable(os.Stdout, res.Generations)
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
	f.StringVar(&typeKind, "type", "", "filter by type: music|instrumental")
	f.BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout")
	return cmd
}
