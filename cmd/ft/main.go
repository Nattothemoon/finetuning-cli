// Command ft is the Finetuning AI CLI.
package main

import (
	"fmt"
	"os"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/commands"
)

// Set at build time via -ldflags "-X main.version=v1.2.3 -X main.commit=... -X main.date=..."
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	api.Version = version
	cmd := commands.NewRootCmd()
	cmd.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor maps APIError codes to the codes the spec asks us to use.
func exitCodeFor(err error) int {
	if apiErr, ok := api.IsAPIError(err); ok {
		switch apiErr.Code {
		case "VALIDATION_ERROR":
			return 2
		}
		// Any other API error → exit 1 with the server's message.
		return 1
	}
	return 1
}
