package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/garygentry/dotfiles/internal/module"
)

// ProgressTracker tracks the progress of module installation.
type ProgressTracker struct {
	total       int
	current     int
	startTime   time.Time
	moduleTimes []time.Duration
	writer      io.Writer
	isTTY       bool
}

// StartProgressBar creates and displays a new progress tracker.
func (u *UI) StartProgressBar(total int) module.ProgressTracker {
	return &ProgressTracker{
		total:       total,
		current:     0,
		startTime:   time.Now(),
		moduleTimes: make([]time.Duration, 0, total),
		writer:      u.writer,
		isTTY:       u.IsTTY,
	}
}

// UpdateProgress updates the progress bar with the current module.
func (u *UI) UpdateProgress(handle module.ProgressTracker, current int, moduleName string) {
	if handle == nil {
		return
	}

	p, ok := handle.(*ProgressTracker)
	if !ok {
		return
	}

	p.current = current

	if !p.isTTY {
		// Non-TTY: simple text progress
		fmt.Fprintf(p.writer, "[PROGRESS] %d/%d: %s\n", current, p.total, moduleName)
		return
	}

	// TTY: render progress bar
	elapsed := time.Since(p.startTime)
	percentage := 0
	if p.total > 0 {
		percentage = (current * 100) / p.total
	}

	// Estimate remaining time based on average module duration
	var remaining time.Duration
	if current > 0 && len(p.moduleTimes) > 0 {
		var totalDuration time.Duration
		for _, d := range p.moduleTimes {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(p.moduleTimes))
		remaining = avgDuration * time.Duration(p.total-current)
	}

	// Build progress bar
	barWidth := 40
	filled := (barWidth * current) / p.total
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Style the components
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#89b4fa")). // Blue
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89b4fa")). // Blue
		Bold(true)

	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")) // Green

	subtextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")) // Subtext

	var content strings.Builder
	content.WriteString(titleStyle.Render(fmt.Sprintf("Installing %d modules", p.total)))
	content.WriteString("  ")
	content.WriteString(progressStyle.Render(bar))
	content.WriteString("  ")
	content.WriteString(titleStyle.Render(fmt.Sprintf("%d/%d (%d%%)", current, p.total, percentage)))
	content.WriteString("\n")

	// Second line: current module and timing
	var timingInfo string
	if remaining > 0 {
		timingInfo = fmt.Sprintf("Current: %s • Elapsed: %v • Est. remaining: ~%v",
			moduleName, elapsed.Round(time.Second), remaining.Round(time.Second))
	} else {
		timingInfo = fmt.Sprintf("Current: %s • Elapsed: %v",
			moduleName, elapsed.Round(time.Second))
	}
	content.WriteString(subtextStyle.Render(timingInfo))

	// Print the bordered box (with newline before to separate from previous output)
	fmt.Fprintf(p.writer, "\n%s\n", borderStyle.Render(content.String()))
}

// RecordModuleTime records the duration of a module execution for time estimation.
func (u *UI) RecordModuleTime(handle module.ProgressTracker, duration time.Duration) {
	if handle == nil {
		return
	}

	p, ok := handle.(*ProgressTracker)
	if !ok {
		return
	}

	p.moduleTimes = append(p.moduleTimes, duration)
}

// FinishProgress displays the final summary.
func (u *UI) FinishProgress(handle module.ProgressTracker, summary *module.ProgressSummary) {
	if handle == nil {
		return
	}

	p, ok := handle.(*ProgressTracker)
	if !ok {
		return
	}

	if !p.isTTY {
		// Non-TTY: simple text summary
		fmt.Fprintf(p.writer, "[SUMMARY] Completed in %v: %d succeeded, %d failed, %d skipped\n",
			summary.Duration, summary.Succeeded, summary.Failed, summary.Skipped)
		return
	}

	// TTY: render completion summary
	percentage := 100
	barWidth := 40
	bar := strings.Repeat("█", barWidth)

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#a6e3a1")). // Green
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")). // Green
		Bold(true)

	successIcon := "\u2713" // ✓

	var content strings.Builder
	content.WriteString(titleStyle.Render(fmt.Sprintf("%s Installation complete", successIcon)))
	content.WriteString("  ")
	content.WriteString(titleStyle.Render(bar))
	content.WriteString("  ")
	content.WriteString(titleStyle.Render(fmt.Sprintf("%d/%d (%d%%)", summary.Total, summary.Total, percentage)))
	content.WriteString("\n")

	// Second line: summary stats
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")) // Text

	stats := fmt.Sprintf("Success: %d • Failed: %d • Skipped: %d • Time: %v",
		summary.Succeeded, summary.Failed, summary.Skipped, summary.Duration.Round(time.Second))
	content.WriteString(statsStyle.Render(stats))

	fmt.Fprintf(p.writer, "\n%s\n\n", borderStyle.Render(content.String()))
}
