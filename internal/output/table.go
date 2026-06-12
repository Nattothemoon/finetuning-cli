package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/finetuning/cli/internal/api"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// GenerationTable renders a list of generations as a compact table.
func GenerationTable(w io.Writer, gens []api.Generation) {
	t := newPlainTable(w)
	t.Style().Format.Header = text.FormatUpper
	t.AppendHeader(table.Row{"ID", "STATUS", "DURATION", "PROMPT", "CREATED"})
	for _, g := range gens {
		duration := "—"
		if g.Duration > 0 {
			duration = formatSeconds(g.Duration)
		}
		t.AppendRow(table.Row{
			g.ID,
			string(g.Status),
			duration,
			truncate(g.Prompt, 40),
			HumanTime(g.CreatedAt),
		})
	}
	t.Render()
}

// GenerationDetail renders a single generation as a labelled block.
func GenerationDetail(w io.Writer, g *api.Generation) {
	t := newPlainTable(w)
	add := func(k, v string) { t.AppendRow(table.Row{k, v}) }
	add("ID", g.ID)
	add("Status", string(g.Status))
	if g.Title != "" {
		add("Title", g.Title)
	}
	add("Prompt", g.Prompt)
	if g.Duration > 0 {
		add("Duration", formatSeconds(g.Duration))
	}
	if g.AudioURL != "" {
		add("Audio URL", g.AudioURL)
	}
	if g.FileSize > 0 {
		add("File size", HumanBytes(g.FileSize))
	}
	if g.GenerationTime > 0 {
		add("GPU time", fmt.Sprintf("%.1fs", g.GenerationTime))
	}
	if g.ErrorMessage != "" {
		add("Error", g.ErrorMessage)
	}
	add("Created", HumanTime(g.CreatedAt))
	if g.CompletedAt != "" {
		add("Completed", HumanTime(g.CompletedAt))
	}
	t.Render()
}

// PlaylistTable renders the caller's playlists as a compact table.
func PlaylistTable(w io.Writer, pls []api.Playlist) {
	t := newPlainTable(w)
	t.Style().Format.Header = text.FormatUpper
	t.AppendHeader(table.Row{"ID", "NAME", "TRACKS", "DURATION", "VISIBILITY", "UPDATED"})
	for _, p := range pls {
		duration := "—"
		if p.TotalDuration > 0 {
			duration = formatSeconds(p.TotalDuration)
		}
		visibility := "private"
		if p.IsPublic {
			visibility = "public"
		}
		t.AppendRow(table.Row{
			p.ID,
			truncate(p.Name, 30),
			p.TrackCount,
			duration,
			visibility,
			HumanTime(p.UpdatedAt),
		})
	}
	t.Render()
}

// PlaylistDetailBlock renders one playlist's header; the caller follows up
// with GenerationTable for the tracks.
func PlaylistDetailBlock(w io.Writer, p *api.PlaylistDetail) {
	t := newPlainTable(w)
	add := func(k, v string) { t.AppendRow(table.Row{k, v}) }
	add("ID", p.ID)
	add("Name", p.Name)
	if p.Description != "" {
		add("Description", p.Description)
	}
	visibility := "private"
	if p.IsPublic {
		visibility = "public"
	}
	add("Visibility", visibility)
	add("Tracks", fmt.Sprintf("%d", p.TrackCount))
	if p.TotalDuration > 0 {
		add("Duration", formatSeconds(p.TotalDuration))
	}
	if !p.IsOwner {
		add("Owner", "someone else (public playlist)")
	}
	add("Updated", HumanTime(p.UpdatedAt))
	t.Render()
}

// MeBlock renders /v1/me into a friendly summary on w.
func MeBlock(w io.Writer, me *api.Me) {
	t := newPlainTable(w)
	add := func(k, v string) { t.AppendRow(table.Row{k, v}) }
	add("Email", me.Email)
	if me.Name != "" {
		add("Name", me.Name)
	}
	add("Tier", me.Tier)
	add("Monthly remaining", fmt.Sprintf("%d / %d", me.Limits.MonthlyRemaining, me.Limits.MonthlyGenerations))
	add("Pack credits", fmt.Sprintf("%d", me.Limits.PackCredits))
	add("Queue depth", fmt.Sprintf("%d", me.Limits.QueueDepth))
	t.Render()
}

func newPlainTable(w io.Writer) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	style := t.Style()
	style.Options.DrawBorder = false
	style.Options.SeparateColumns = false
	style.Options.SeparateRows = false
	style.Options.SeparateHeader = false
	style.Options.SeparateFooter = false
	return t
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func formatSeconds(s int) string {
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m, r := s/60, s%60
	if r == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, r)
}
