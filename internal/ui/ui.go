package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/garygentry/dotfiles/internal/module"
)

// Catppuccin Mocha color palette as ANSI RGB escape sequences.
var (
	colorBlue    = "\033[38;2;137;180;250m"
	colorGreen   = "\033[38;2;166;227;161m"
	colorRed     = "\033[38;2;243;139;168m"
	colorYellow  = "\033[38;2;249;226;175m"
	colorMauve   = "\033[38;2;203;166;247m"
	colorText    = "\033[38;2;205;214;244m"
	colorSubtext = "\033[38;2;166;173;200m"
	colorSurface = "\033[38;2;49;50;68m"
	colorReset   = "\033[0m"
)

// Log level prefix icons.
const (
	iconInfo    = "\u2022" // •
	iconWarn    = "\u26a0" // ⚠
	iconError   = "\u2717" // ✗
	iconSuccess = "\u2713" // ✓
	iconDebug   = "\u2192" // →
)

// Braille spinner frames, rotating at 80ms per frame.
var spinnerFrames = []string{
	"\u280b", // ⠋
	"\u2819", // ⠙
	"\u2839", // ⠹
	"\u2838", // ⠸
	"\u283c", // ⠼
	"\u2834", // ⠴
	"\u2826", // ⠦
	"\u2827", // ⠧
	"\u2807", // ⠇
	"\u280f", // ⠏
}

// Spinner displays an animated braille spinner in the terminal.
type Spinner struct {
	message string
	done    chan struct{}
	active  bool
	mu      sync.Mutex
}

// UI provides styled terminal output for the dotfiles CLI.
type UI struct {
	Verbose bool
	IsTTY   bool
	writer  io.Writer
}

// New creates a new UI instance. It detects whether stdout is a TTY and stores
// the verbose flag. Output is written to os.Stdout by default.
func New(verbose bool) *UI {
	isTTY := isTerminal(os.Stdout)
	return &UI{
		Verbose: verbose,
		IsTTY:   isTTY,
		writer:  os.Stdout,
	}
}

// NewWithWriter creates a UI that writes to a custom writer. This is primarily
// useful for testing. The IsTTY field is set to the provided value.
func NewWithWriter(writer io.Writer, verbose bool, isTTY bool) *UI {
	return &UI{
		Verbose: verbose,
		IsTTY:   isTTY,
		writer:  writer,
	}
}

// isTerminal checks whether the given file is connected to a terminal.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// --- Log methods ---

// Info prints an informational message with a blue bullet prefix.
func (u *UI) Info(msg string) {
	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s%s\n", colorBlue, iconInfo, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[INFO] %s\n", msg)
	}
}

// Warn prints a warning message with a yellow warning icon prefix.
func (u *UI) Warn(msg string) {
	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s%s\n", colorYellow, iconWarn, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[WARN] %s\n", msg)
	}
}

// Error prints an error message with a red cross prefix.
func (u *UI) Error(msg string) {
	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s%s\n", colorRed, iconError, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[ERROR] %s\n", msg)
	}
}

// Success prints a success message with a green checkmark prefix.
func (u *UI) Success(msg string) {
	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s%s\n", colorGreen, iconSuccess, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[OK] %s\n", msg)
	}
}

// Debug prints a debug message with a mauve arrow prefix. The message is only
// printed when verbose mode is enabled.
func (u *UI) Debug(msg string) {
	if !u.Verbose {
		return
	}
	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s%s\n", colorMauve, iconDebug, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[DEBUG] %s\n", msg)
	}
}

// --- Spinner methods ---

// StartSpinner begins an animated braille spinner with the given message.
// In non-TTY mode the message is printed as a plain info line instead.
// Returns an opaque handle (of type any) for use with StopSpinner* methods.
func (u *UI) StartSpinner(msg string) any {
	s := &Spinner{
		message: msg,
		done:    make(chan struct{}),
		active:  true,
	}

	if !u.IsTTY {
		fmt.Fprintf(u.writer, "[INFO] %s...\n", msg)
		return s
	}

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				frame := spinnerFrames[i%len(spinnerFrames)]
				fmt.Fprintf(u.writer, "\r%s%s %s%s", colorMauve, frame, msg, colorReset)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return s
}

// StopSpinnerSuccess stops the spinner and prints a success message.
func (u *UI) StopSpinnerSuccess(s any, msg string) {
	u.stopSpinner(s)
	if u.IsTTY {
		fmt.Fprintf(u.writer, "\r\033[K%s%s %s%s\n", colorGreen, iconSuccess, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[OK] %s\n", msg)
	}
}

// StopSpinnerFail stops the spinner and prints a failure message.
func (u *UI) StopSpinnerFail(s any, msg string) {
	u.stopSpinner(s)
	if u.IsTTY {
		fmt.Fprintf(u.writer, "\r\033[K%s%s %s%s\n", colorRed, iconError, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[ERROR] %s\n", msg)
	}
}

// StopSpinnerSkip stops the spinner and prints a skip message.
func (u *UI) StopSpinnerSkip(s any, msg string) {
	u.stopSpinner(s)
	if u.IsTTY {
		fmt.Fprintf(u.writer, "\r\033[K%s%s %s%s\n", colorYellow, iconWarn, msg, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[SKIP] %s\n", msg)
	}
}

// stopSpinner signals the spinner goroutine to stop.
func (u *UI) stopSpinner(handle any) {
	if handle == nil {
		return
	}
	s, ok := handle.(*Spinner)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		s.active = false
		close(s.done)
		// Small sleep to let the goroutine exit cleanly.
		time.Sleep(100 * time.Millisecond)
	}
}

// --- Prompt methods ---

// PromptInput asks the user for text input. If the user enters nothing, the
// default value is returned. Returns an error if reading from stdin fails.
func (u *UI) PromptInput(msg string, defaultVal string) (string, error) {
	prompt := msg
	if defaultVal != "" {
		prompt = fmt.Sprintf("%s [%s]", msg, defaultVal)
	}

	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s:%s ", colorBlue, iconInfo, prompt, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[INPUT] %s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// PromptConfirm asks the user a yes/no question. The default value is used
// when the user presses enter without typing anything.
func (u *UI) PromptConfirm(msg string, defaultVal bool) (bool, error) {
	hint := "y/N"
	if defaultVal {
		hint = "Y/n"
	}

	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s [%s]:%s ", colorBlue, iconInfo, msg, hint, colorReset)
	} else {
		fmt.Fprintf(u.writer, "[CONFIRM] %s [%s]: ", msg, hint)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "":
		return defaultVal, nil
	default:
		return defaultVal, nil
	}
}

// PromptChoice presents a numbered list of options and asks the user to pick
// one. Returns the selected option string.
func (u *UI) PromptChoice(msg string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	if u.IsTTY {
		fmt.Fprintf(u.writer, "%s%s %s%s\n", colorBlue, iconInfo, msg, colorReset)
		for i, opt := range options {
			fmt.Fprintf(u.writer, "  %s%d)%s %s\n", colorMauve, i+1, colorReset, opt)
		}
		fmt.Fprintf(u.writer, "%s%s Choice [1-%d]:%s ", colorBlue, iconInfo, len(options), colorReset)
	} else {
		fmt.Fprintf(u.writer, "[CHOICE] %s\n", msg)
		for i, opt := range options {
			fmt.Fprintf(u.writer, "  %d) %s\n", i+1, opt)
		}
		fmt.Fprintf(u.writer, "[CHOICE] Pick [1-%d]: ", len(options))
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil {
		return "", fmt.Errorf("invalid choice: %q", input)
	}

	if choice < 1 || choice > len(options) {
		return "", fmt.Errorf("choice out of range: %d", choice)
	}

	return options[choice-1], nil
}

// --- Execution plan ---

// PrintExecutionPlan displays a formatted execution plan showing which modules
// will be installed and which will be skipped.
func (u *UI) PrintExecutionPlan(modules []*module.Module, skipped []*module.Module) {
	if u.IsTTY {
		u.printExecutionPlanTTY(modules, skipped)
	} else {
		u.printExecutionPlanPlain(modules, skipped)
	}
}

func (u *UI) printExecutionPlanTTY(modules []*module.Module, skipped []*module.Module) {
	fmt.Fprintf(u.writer, "\n%s%s Execution Plan%s\n", colorBlue, iconInfo, colorReset)
	fmt.Fprintf(u.writer, "%s%s%s\n", colorSurface, strings.Repeat("\u2500", 40), colorReset)

	if len(modules) > 0 {
		fmt.Fprintf(u.writer, "\n  %sInstall (%d):%s\n", colorGreen, len(modules), colorReset)
		for i, m := range modules {
			desc := m.Description
			if desc == "" {
				desc = "no description"
			}
			fmt.Fprintf(u.writer, "  %s%d.%s %s%s%s %s- %s%s\n",
				colorMauve, i+1, colorReset,
				colorText, m.Name, colorReset,
				colorSubtext, desc, colorReset,
			)
		}
	}

	if len(skipped) > 0 {
		fmt.Fprintf(u.writer, "\n  %sSkipped (%d):%s\n", colorYellow, len(skipped), colorReset)
		for _, m := range skipped {
			desc := m.Description
			if desc == "" {
				desc = "no description"
			}
			fmt.Fprintf(u.writer, "  %s%s%s %s%s%s %s- %s%s\n",
				colorYellow, iconWarn, colorReset,
				colorSubtext, m.Name, colorReset,
				colorSubtext, desc, colorReset,
			)
		}
	}

	fmt.Fprintf(u.writer, "\n%s%s%s\n\n", colorSurface, strings.Repeat("\u2500", 40), colorReset)
}

func (u *UI) printExecutionPlanPlain(modules []*module.Module, skipped []*module.Module) {
	fmt.Fprintf(u.writer, "\n[INFO] Execution Plan\n")
	fmt.Fprintf(u.writer, "%s\n", strings.Repeat("-", 40))

	if len(modules) > 0 {
		fmt.Fprintf(u.writer, "\n  Install (%d):\n", len(modules))
		for i, m := range modules {
			desc := m.Description
			if desc == "" {
				desc = "no description"
			}
			fmt.Fprintf(u.writer, "  %d. %s - %s\n", i+1, m.Name, desc)
		}
	}

	if len(skipped) > 0 {
		fmt.Fprintf(u.writer, "\n  Skipped (%d):\n", len(skipped))
		for _, m := range skipped {
			desc := m.Description
			if desc == "" {
				desc = "no description"
			}
			fmt.Fprintf(u.writer, "  [SKIP] %s - %s\n", m.Name, desc)
		}
	}

	fmt.Fprintf(u.writer, "\n%s\n\n", strings.Repeat("-", 40))
}
