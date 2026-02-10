package dotfiles

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestStatusCommandHelp(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of installed modules",
		Long:  statusCmd.Long,
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	requiredStrings := []string{
		"status",
		"modules",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(required)) {
			t.Errorf("Help output missing expected string %q.\nOutput: %s", required, output)
		}
	}
}

func TestFormatTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		wantSubs string // substring that should be present
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			wantSubs: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			wantSubs: "1 min ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			wantSubs: "5 mins ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			wantSubs: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			wantSubs: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			wantSubs: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			wantSubs: "3 days ago",
		},
		{
			name:     "old date format",
			time:     now.Add(-30 * 24 * time.Hour),
			wantSubs: "-", // Should contain date format YYYY-MM-DD
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.time)
			if !strings.Contains(got, tt.wantSubs) {
				t.Errorf("formatTime(%v) = %q, want to contain %q", tt.time, got, tt.wantSubs)
			}
		})
	}
}

func TestFormatTimeEdgeCases(t *testing.T) {
	// Test exact boundaries
	now := time.Now()

	tests := []struct {
		name string
		time time.Time
	}{
		{
			name: "exactly 1 minute",
			time: now.Add(-1 * time.Minute),
		},
		{
			name: "exactly 1 hour",
			time: now.Add(-1 * time.Hour),
		},
		{
			name: "exactly 1 day",
			time: now.Add(-24 * time.Hour),
		},
		{
			name: "exactly 7 days",
			time: now.Add(-7 * 24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.time)
			// Just verify it doesn't panic and returns non-empty string
			if result == "" {
				t.Errorf("formatTime() returned empty string for %v", tt.time)
			}
		})
	}
}

func TestFormatTimeFutureTime(t *testing.T) {
	// Test with future time (edge case)
	future := time.Now().Add(5 * time.Minute)
	result := formatTime(future)

	// Should not panic, but behavior is undefined
	// We just verify it returns something
	if result == "" {
		t.Error("formatTime() returned empty string for future time")
	}
}

func TestFormatTimeZeroValue(t *testing.T) {
	// Test with zero time value
	zero := time.Time{}
	result := formatTime(zero)

	// Should return a date format (since it's very old)
	if !strings.Contains(result, "-") {
		t.Errorf("formatTime(zero) = %q, expected date format with dashes", result)
	}
}
