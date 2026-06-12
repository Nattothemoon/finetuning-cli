package commands

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

// chunkStrings splits ids into slices of at most n — the server caps bulk
// requests at api.BulkLimit ids, so callers loop chunks and merge results.
func chunkStrings(ids []string, n int) [][]string {
	var chunks [][]string
	for len(ids) > n {
		chunks = append(chunks, ids[:n])
		ids = ids[n:]
	}
	return append(chunks, ids)
}

// confirm asks a y/N question on stderr and reads the answer from stdin.
func confirm(prompt string) (bool, error) {
	if !output.StdinIsTTY() {
		return false, fmt.Errorf("stdin is not a terminal — pass --yes to skip the confirmation")
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}

// bulkOutcome prints per-item failures as warnings and a one-line summary.
// It returns an error (→ non-zero exit) only when nothing succeeded, mirroring
// the web client's bulk-select behavior.
func bulkOutcome(verbPast string, succeeded []string, errs []api.ItemError) error {
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", e.Key(), e.Error)
	}
	privacyHint(errs)
	total := len(succeeded) + len(errs)
	if len(succeeded) == 0 {
		return fmt.Errorf("%s 0 of %d track(s)", strings.ToLower(verbPast), total)
	}
	fmt.Printf("%s %d of %d track(s)\n", verbPast, len(succeeded), total)
	return nil
}

// privacyHint explains the one failure the CLI can't work around: /v1 has no
// endpoint to flip track visibility, so the user must do it in the web app.
func privacyHint(errs []api.ItemError) {
	for _, e := range errs {
		if e.Error == "Cannot add private track to public playlist" {
			fmt.Fprintln(os.Stderr, "hint: make the track public at https://finetuning.ai first — the API can't change track visibility.")
			return
		}
	}
}

// looksLikePlaylistID reports whether ref is shaped like a playlist id: the
// documented pl_ prefix, or a UUID (what the live API actually returns).
func looksLikePlaylistID(ref string) bool {
	if strings.HasPrefix(ref, "pl_") {
		return true
	}
	return uuidShape.MatchString(ref)
}

var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// resolvePlaylist turns a playlist reference (id or name) into an id. Ids
// pass through untouched; anything else is matched case-insensitively
// against the caller's playlist names.
func resolvePlaylist(cmd *cobra.Command, ref string) (string, error) {
	if looksLikePlaylistID(ref) {
		return ref, nil
	}
	pls, err := getClient(cmd).ListPlaylists(cmd.Context())
	if err != nil {
		return "", err
	}
	var matches []api.Playlist
	for _, p := range pls {
		if strings.EqualFold(p.Name, ref) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no playlist named %q — run `ft playlists` to see yours", ref)
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		return "", fmt.Errorf("playlist name %q is ambiguous (%s) — use the id instead", ref, strings.Join(ids, ", "))
	}
}
