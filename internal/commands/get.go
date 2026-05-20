package commands

import (
	"os"

	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one generation's detail",
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
			if jsonOut {
				return output.RawJSON(os.Stdout, g)
			}
			output.GenerationDetail(os.Stdout, g)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout")
	return cmd
}
