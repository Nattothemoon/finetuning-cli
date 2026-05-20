package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/auth"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage your API key",
	}
	cmd.AddCommand(newAuthLoginCmd(), newAuthLogoutCmd(), newAuthWhoamiCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var fromStdin bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an API key in the OS keychain (or fallback file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := readKey(fromStdin)
			if err != nil {
				return err
			}
			if !api.ValidateKey(key) {
				return fmt.Errorf("invalid key — expected ft_live_ prefix and length 40, got %d chars", len(key))
			}
			source, err := auth.Set(key)
			if err != nil {
				return fmt.Errorf("store key: %w", err)
			}
			// Verify by hitting /v1/me — it costs no credits.
			client := api.NewClient(getConfig(cmd).BaseURL, key)
			me, err := client.Me(cmd.Context())
			if err != nil {
				return fmt.Errorf("key stored but verification failed: %w", err)
			}
			location := storeLocation(source)
			fmt.Fprintf(os.Stderr, "Stored in %s. ✓\n", location)
			fmt.Fprintf(os.Stderr, "Signed in as %s (%s tier)\n", me.Email, me.Tier)
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read key from stdin instead of prompting")
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Forget the stored API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Delete(); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Logged out.")
			return nil
		},
	}
}

func newAuthWhoamiCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the signed-in account",
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

// readKey reads the API key from stdin (piped) or prompts the user with echo off.
func readKey(forceStdin bool) (string, error) {
	stdinFd := int(os.Stdin.Fd())
	isTTY := term.IsTerminal(stdinFd)
	if forceStdin || !isTTY {
		// Pipe / redirect: read one line.
		s := bufio.NewScanner(os.Stdin)
		if !s.Scan() {
			if err := s.Err(); err != nil {
				return "", err
			}
			return "", errors.New("no input on stdin")
		}
		return strings.TrimSpace(s.Text()), nil
	}
	fmt.Fprint(os.Stderr, "Paste your API key: ")
	buf, err := term.ReadPassword(stdinFd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}

func storeLocation(s auth.Source) string {
	switch s {
	case auth.SourceKeychain:
		return "OS keychain"
	case auth.SourceFile:
		return "plaintext file (~/.config/finetuning/credentials)"
	default:
		return string(s)
	}
}
