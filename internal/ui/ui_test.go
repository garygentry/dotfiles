package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/garygentry/dotfiles/internal/module"
)

func TestNewReturnsNonNil(t *testing.T) {
	u := New(false)
	if u == nil {
		t.Fatal("expected New() to return a non-nil *UI")
	}
}

func TestNewSetsVerbose(t *testing.T) {
	u := New(true)
	if !u.Verbose {
		t.Error("expected Verbose to be true")
	}

	u = New(false)
	if u.Verbose {
		t.Error("expected Verbose to be false")
	}
}

func TestNewSetsWriter(t *testing.T) {
	u := New(false)
	if u.writer == nil {
		t.Error("expected writer to be set")
	}
}

// --- TTY mode log methods ---

func TestInfoTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, true)
	u.Info("hello world")
	out := buf.String()
	if !strings.Contains(out, "\u2022") {
		t.Errorf("expected info icon in output, got: %q", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "\033[38;2;137;180;250m") {
		t.Errorf("expected blue color code in output, got: %q", out)
	}
}

func TestWarnTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, true)
	u.Warn("careful now")
	out := buf.String()
	if !strings.Contains(out, "\u26a0") {
		t.Errorf("expected warn icon in output, got: %q", out)
	}
	if !strings.Contains(out, "careful now") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "\033[38;2;249;226;175m") {
		t.Errorf("expected yellow color code in output, got: %q", out)
	}
}

func TestErrorTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, true)
	u.Error("something broke")
	out := buf.String()
	if !strings.Contains(out, "\u2717") {
		t.Errorf("expected error icon in output, got: %q", out)
	}
	if !strings.Contains(out, "something broke") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "\033[38;2;243;139;168m") {
		t.Errorf("expected red color code in output, got: %q", out)
	}
}

func TestSuccessTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, true)
	u.Success("all good")
	out := buf.String()
	if !strings.Contains(out, "\u2713") {
		t.Errorf("expected success icon in output, got: %q", out)
	}
	if !strings.Contains(out, "all good") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "\033[38;2;166;227;161m") {
		t.Errorf("expected green color code in output, got: %q", out)
	}
}

func TestDebugTTYVerbose(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, true, true)
	u.Debug("debug info")
	out := buf.String()
	if !strings.Contains(out, "\u2192") {
		t.Errorf("expected debug icon in output, got: %q", out)
	}
	if !strings.Contains(out, "debug info") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "\033[38;2;203;166;247m") {
		t.Errorf("expected mauve color code in output, got: %q", out)
	}
}

func TestDebugSilentWhenNotVerbose(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, true)
	u.Debug("should not appear")
	out := buf.String()
	if out != "" {
		t.Errorf("expected no output when Verbose is false, got: %q", out)
	}
}

// --- Non-TTY mode log methods ---

func TestInfoNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	u.Info("hello non-tty")
	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] prefix in non-TTY mode, got: %q", out)
	}
	if !strings.Contains(out, "hello non-tty") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI escape codes in non-TTY mode, got: %q", out)
	}
}

func TestWarnNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	u.Warn("warn non-tty")
	out := buf.String()
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("expected [WARN] prefix in non-TTY mode, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI escape codes in non-TTY mode, got: %q", out)
	}
}

func TestErrorNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	u.Error("error non-tty")
	out := buf.String()
	if !strings.Contains(out, "[ERROR]") {
		t.Errorf("expected [ERROR] prefix in non-TTY mode, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI escape codes in non-TTY mode, got: %q", out)
	}
}

func TestSuccessNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	u.Success("ok non-tty")
	out := buf.String()
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected [OK] prefix in non-TTY mode, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI escape codes in non-TTY mode, got: %q", out)
	}
}

func TestDebugNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, true, false)
	u.Debug("debug non-tty")
	out := buf.String()
	if !strings.Contains(out, "[DEBUG]") {
		t.Errorf("expected [DEBUG] prefix in non-TTY mode, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI escape codes in non-TTY mode, got: %q", out)
	}
}

// --- Non-TTY mode strips colors across all methods ---

func TestNonTTYStripsColors(t *testing.T) {
	methods := []struct {
		name string
		call func(u *UI)
	}{
		{"Info", func(u *UI) { u.Info("msg") }},
		{"Warn", func(u *UI) { u.Warn("msg") }},
		{"Error", func(u *UI) { u.Error("msg") }},
		{"Success", func(u *UI) { u.Success("msg") }},
		{"Debug", func(u *UI) { u.Debug("msg") }},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			var buf bytes.Buffer
			u := NewWithWriter(&buf, true, false)
			m.call(u)
			out := buf.String()
			if strings.Contains(out, "\033[") {
				t.Errorf("%s: expected no ANSI escape codes in non-TTY mode, got: %q", m.name, out)
			}
		})
	}
}

// --- Spinner in non-TTY mode ---

func TestSpinnerNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	s := u.StartSpinner("loading")
	out := buf.String()
	if !strings.Contains(out, "[INFO] loading...") {
		t.Errorf("expected plain spinner start message, got: %q", out)
	}

	buf.Reset()
	u.StopSpinnerSuccess(s, "done loading")
	out = buf.String()
	if !strings.Contains(out, "[OK] done loading") {
		t.Errorf("expected plain success message, got: %q", out)
	}
}

func TestSpinnerFailNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	s := u.StartSpinner("loading")
	buf.Reset()
	u.StopSpinnerFail(s, "failed to load")
	out := buf.String()
	if !strings.Contains(out, "[ERROR] failed to load") {
		t.Errorf("expected plain fail message, got: %q", out)
	}
}

func TestSpinnerSkipNonTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)
	s := u.StartSpinner("loading")
	buf.Reset()
	u.StopSpinnerSkip(s, "skipped")
	out := buf.String()
	if !strings.Contains(out, "[SKIP] skipped") {
		t.Errorf("expected plain skip message, got: %q", out)
	}
}

// --- Execution plan ---

func TestPrintExecutionPlanTTY(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, true)

	modules := []*module.Module{
		{Name: "zsh", Description: "Zsh shell configuration"},
		{Name: "git", Description: "Git configuration"},
	}
	skipped := []*module.Module{
		{Name: "macos", Description: "macOS-specific settings"},
	}

	u.PrintExecutionPlan(modules, skipped)
	out := buf.String()

	if !strings.Contains(out, "Execution Plan") {
		t.Errorf("expected 'Execution Plan' header, got: %q", out)
	}
	if !strings.Contains(out, "Install (2)") {
		t.Errorf("expected 'Install (2)' section, got: %q", out)
	}
	if !strings.Contains(out, "zsh") {
		t.Errorf("expected module name 'zsh', got: %q", out)
	}
	if !strings.Contains(out, "git") {
		t.Errorf("expected module name 'git', got: %q", out)
	}
	if !strings.Contains(out, "Skipped (1)") {
		t.Errorf("expected 'Skipped (1)' section, got: %q", out)
	}
	if !strings.Contains(out, "macos") {
		t.Errorf("expected module name 'macos', got: %q", out)
	}
}

func TestPrintExecutionPlanPlain(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)

	modules := []*module.Module{
		{Name: "vim", Description: "Vim editor configuration"},
	}
	var skipped []*module.Module

	u.PrintExecutionPlan(modules, skipped)
	out := buf.String()

	if !strings.Contains(out, "[INFO] Execution Plan") {
		t.Errorf("expected '[INFO] Execution Plan' header, got: %q", out)
	}
	if !strings.Contains(out, "Install (1)") {
		t.Errorf("expected 'Install (1)' section, got: %q", out)
	}
	if !strings.Contains(out, "vim") {
		t.Errorf("expected module name 'vim', got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI escape codes in plain mode, got: %q", out)
	}
}

func TestPrintExecutionPlanNoDescription(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, false, false)

	modules := []*module.Module{
		{Name: "empty"},
	}

	u.PrintExecutionPlan(modules, nil)
	out := buf.String()

	if !strings.Contains(out, "no description") {
		t.Errorf("expected 'no description' fallback, got: %q", out)
	}
}
