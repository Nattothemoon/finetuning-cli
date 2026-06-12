package commands

import (
	"fmt"
	"os"

	"github.com/finetuning/cli/internal/api"
	"github.com/finetuning/cli/internal/output"
	"github.com/spf13/cobra"
)

func newPlaylistsCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "playlists",
		Short: "List your playlists",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			pls, err := c.ListPlaylists(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, map[string]any{"playlists": pls})
			}
			if len(pls) == 0 {
				fmt.Fprintln(os.Stderr, "No playlists yet — create one in the web app at https://finetuning.ai.")
				return nil
			}
			output.PlaylistTable(os.Stdout, pls)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit raw JSON to stdout")
	return cmd
}

func newPlaylistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playlist",
		Short: "Manage playlist tracks (add, remove, move)",
	}
	cmd.AddCommand(
		newPlaylistAddCmd(),
		newPlaylistRemoveCmd(),
		newPlaylistMoveCmd(),
	)
	return cmd
}

func newPlaylistAddCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "add <playlist> <track-id>...",
		Short: "Add tracks to a playlist",
		Long:  "Adds tracks to a playlist. <playlist> is an id (pl_...) or a playlist name.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			playlistID, err := resolvePlaylist(cmd, args[0])
			if err != nil {
				return err
			}
			merged := &api.AddTracksResult{Added: []string{}}
			for _, chunk := range chunkStrings(args[1:], api.BulkLimit) {
				res, err := c.AddPlaylistTracks(cmd.Context(), playlistID, chunk)
				if err != nil {
					return err
				}
				merged.Added = append(merged.Added, res.Added...)
				merged.Errors = append(merged.Errors, res.Errors...)
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, merged)
			}
			return bulkOutcome("Added", merged.Added, merged.Errors)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit merged raw JSON to stdout")
	return cmd
}

func newPlaylistRemoveCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "remove <playlist> <track-id>...",
		Short: "Remove tracks from a playlist",
		Long:  "Removes tracks from a playlist. The tracks stay in your library. <playlist> is an id (pl_...) or a playlist name.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			playlistID, err := resolvePlaylist(cmd, args[0])
			if err != nil {
				return err
			}
			merged := &api.RemoveTracksResult{Removed: []string{}}
			for _, chunk := range chunkStrings(args[1:], api.BulkLimit) {
				res, err := c.RemovePlaylistTracks(cmd.Context(), playlistID, chunk)
				if err != nil {
					return err
				}
				merged.Removed = append(merged.Removed, res.Removed...)
				merged.Errors = append(merged.Errors, res.Errors...)
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, merged)
			}
			return bulkOutcome("Removed", merged.Removed, merged.Errors)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit merged raw JSON to stdout")
	return cmd
}

func newPlaylistMoveCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "move <source> <target> <track-id>...",
		Short: "Move tracks between playlists",
		Long:  "Moves tracks from <source> to <target>. A track that fails to add to the target stays in the source. Both playlists may be an id (pl_...) or a name.",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient(cmd)
			if err := requireKey(c); err != nil {
				return err
			}
			sourceID, err := resolvePlaylist(cmd, args[0])
			if err != nil {
				return err
			}
			targetID, err := resolvePlaylist(cmd, args[1])
			if err != nil {
				return err
			}
			if sourceID == targetID {
				return fmt.Errorf("source and target playlist are the same (%s)", sourceID)
			}
			merged := &api.MoveTracksResult{Moved: []string{}}
			for _, chunk := range chunkStrings(args[2:], api.BulkLimit) {
				res, err := c.MovePlaylistTracks(cmd.Context(), sourceID, targetID, chunk)
				if err != nil {
					return err
				}
				merged.Moved = append(merged.Moved, res.Moved...)
				merged.Errors = append(merged.Errors, res.Errors...)
			}
			if jsonOut {
				return output.RawJSON(os.Stdout, merged)
			}
			return bulkOutcome("Moved", merged.Moved, merged.Errors)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit merged raw JSON to stdout")
	return cmd
}
