package cmd

import (
	"testing"
	"time"
)

func TestParseDue(t *testing.T) {
	// Friday, 2026-04-24 at 10:00 local.
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local)

	tests := []struct {
		in   string
		want time.Time
	}{
		{"today", time.Date(2026, 4, 24, 0, 0, 0, 0, time.Local)},
		{"today 4pm", time.Date(2026, 4, 24, 16, 0, 0, 0, time.Local)},
		{"today 16:00", time.Date(2026, 4, 24, 16, 0, 0, 0, time.Local)},
		{"tomorrow", time.Date(2026, 4, 25, 0, 0, 0, 0, time.Local)},
		{"tomorrow 2pm", time.Date(2026, 4, 25, 14, 0, 0, 0, time.Local)},
		{"tomorrow 2 pm", time.Date(2026, 4, 25, 14, 0, 0, 0, time.Local)},
		{"tonight", time.Date(2026, 4, 24, 20, 0, 0, 0, time.Local)},
		{"2026-04-30", time.Date(2026, 4, 30, 0, 0, 0, 0, time.Local)},
		{"2026-04-30 14:00", time.Date(2026, 4, 30, 14, 0, 0, 0, time.Local)},
		{"30.04.2026", time.Date(2026, 4, 30, 0, 0, 0, 0, time.Local)},
		// Friday is today → Monday is 3 days out.
		{"monday", time.Date(2026, 4, 27, 0, 0, 0, 0, time.Local)},
		{"mon 9am", time.Date(2026, 4, 27, 9, 0, 0, 0, time.Local)},
		// "next friday" when today is Friday → +7 days.
		{"next friday", time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)},
		// "friday" when today is Friday → today.
		{"friday", time.Date(2026, 4, 24, 0, 0, 0, 0, time.Local)},
		{"in 2h", now.Add(2 * time.Hour)},
		{"in 30 minutes", now.Add(30 * time.Minute)},
		{"+3d", now.Add(3 * 24 * time.Hour)},
		{"2h", now.Add(2 * time.Hour)},
		// Bare time before now → bumps to tomorrow.
		{"8am", time.Date(2026, 4, 25, 8, 0, 0, 0, time.Local)},
		// Bare time after now → today.
		{"2pm", time.Date(2026, 4, 24, 14, 0, 0, 0, time.Local)},
	}

	for _, tc := range tests {
		got, err := parseDueAt(tc.in, now)
		if err != nil {
			t.Errorf("parseDueAt(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("parseDueAt(%q): got %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestParseDueErrors(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.Local)
	for _, in := range []string{"", "garbage", "25:99", "tomorrow 99pm"} {
		if _, err := parseDueAt(in, now); err == nil {
			t.Errorf("parseDueAt(%q): expected error, got nil", in)
		}
	}
}
