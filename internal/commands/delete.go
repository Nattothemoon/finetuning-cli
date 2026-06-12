package commands

import (
	"fmt"
	"os"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var (
		yes     bool
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "delete <id>...",
		Short: "Permanently delete tracks from your library",
		Long:  "Permanently deletes tracks. They disappear from your library, playlists, and public pages immediately and cannot be restored via the API.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			if !yes {
				fmt.Fprintln(os.Stderr, "This permanently deletes the track from your library — everywhere, not just playlists.")
				ok, err := confirm(fmt.Sprintf("Delete %d track(s)?", len(args)))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			merged := &api.BulkDeleteResult{Deleted: []string{}}
			for _, chunk := range chunkStrings(args, api.BulkLimit) {
				res, err := c.BulkDeleteGenerations(cmd.Context(), chunk)
				if err != nil {
					return err
				}
				merged.Deleted = append(merged.Deleted, res.Deleted...)
				merged.Errors = append(merged.Errors, res.Errors...)
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, merged)
			}
			return bulkOutcome("Deleted", merged.Deleted, merged.Errors)
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	f.BoolVar(&jsonOut, "json", false, "emit merged raw JSON to stdout")
	return cmd
}
