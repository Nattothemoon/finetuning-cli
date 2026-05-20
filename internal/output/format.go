// Package output centralizes how the CLI renders results. Two principles:
//
//	1. Data → stdout, everything else (spinners, prompts, progress) → stderr.
//	2. JSON mode is byte-faithful to the API: callers pass `any` and we
//	   wrap it in {"data": ...} only if the caller asks us to.
//
// TTY detection is centralized here so commands don't reach into golang.org/x/term.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// StderrIsTTY returns true if stderr is attached to a terminal — used to decide
// whether spinners and progress bars make sense.
func StderrIsTTY() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}

// StdoutIsTTY returns true if stdout is attached to a terminal.
func StdoutIsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// JSON writes `v` as pretty JSON to w (stdout by default).
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// RawJSON re-marshals an arbitrary value as compact-but-readable JSON.
// Convenient when the API gave us a struct and we want the {"data": ...}
// envelope preserved for jq-piping.
func RawJSON(w io.Writer, v any) error {
	return JSON(w, map[string]any{"data": v})
}

// HumanTime renders an API timestamp as a relative phrase ("2 hours ago").
// Falls back to the original string if no known format parses.
//
// The API is inconsistent: `completedAt` ships as RFC3339Nano, while
// `createdAt` ships as `"2006-01-02 15:04:05"` (UTC, no T, no zone).
// We try them in order from most-specific to most-permissive.
func HumanTime(ts string) string {
	if ts == "" {
		return "—"
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05", // API's createdAt — assumed UTC
	}
	var (
		t   time.Time
		err error
	)
	for _, layout := range formats {
		if layout == "2006-01-02 15:04:05" {
			t, err = time.ParseInLocation(layout, ts, time.UTC)
		} else {
			t, err = time.Parse(layout, ts)
		}
		if err == nil {
			break
		}
	}
	if err != nil {
		return ts
	}
	d := time.Since(t)
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

// HumanBytes formats a byte count as "5.2 MB" etc.
func HumanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	v := float64(n) / 1024
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v /= 1024
		idx++
	}
	return fmt.Sprintf("%.1f %s", v, units[idx])
}

// ShortID returns the first 8 chars of a UUID for compact display.
func ShortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
