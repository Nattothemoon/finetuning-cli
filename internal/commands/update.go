package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	installShURL  = "https://raw.githubusercontent.com/Nattothemoon/finetuning-cli/main/scripts/install.sh"
	installPs1URL = "https://raw.githubusercontent.com/Nattothemoon/finetuning-cli/main/scripts/install.ps1"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Re-run the install script to upgrade ft to the latest release",
		Long: "Re-runs the official install script for your OS. The script downloads the latest GitHub release and replaces your `ft` binary.\n\n" +
			"macOS/Linux: requires curl + bash. Windows: runs the PowerShell installer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate()
		},
	}
}

func runUpdate() error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		script := fmt.Sprintf("curl -fsSL %s | bash", installShURL)
		c = exec.Command("bash", "-c", script)
	case "windows":
		script := fmt.Sprintf("iwr -useb %s | iex", installPs1URL)
		c = exec.Command("powershell", "-NoProfile", "-Command", script)
	default:
		return fmt.Errorf("unsupported OS: %s — install manually from https://github.com/Nattothemoon/finetuning-cli/releases", runtime.GOOS)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
