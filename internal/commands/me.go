package commands

import (
	"os"

	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

func newMeCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show signed-in account, tier, and remaining credits (alias of `auth whoami`)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			me, err := c.Me(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, me)
			}
			output.MeBlock(os.Stdout, me)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout")
	return cmd
}
