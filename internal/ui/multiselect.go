package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/garygentry/dotfiles/internal/module"
)

// gridMultiSelect is a compact grid-based multi-select component using Bubble Tea.
type gridMultiSelect struct {
	title       string
	options     []gridOption
	cursor      int     // Linear cursor position (0 to len(options)-1)
	width       int     // Terminal width
	height      int     // Terminal height
	cols        int     // Number of columns in grid
	rows        int     // Number of rows in grid
	selected    map[int]bool
	submitted   bool
	cancelled   bool
}

type gridOption struct {
	value       string
	label       string
	description string
	selected    bool
}

// newGridMultiSelect creates a new grid-based multi-select component.
func newGridMultiSelect(title string, options []module.MultiSelectOption, preSelected []string) *gridMultiSelect {
	// Build selected set for quick lookup
	preSelectedSet := make(map[string]bool)
	for _, v := range preSelected {
		preSelectedSet[v] = true
	}

	// Convert to grid options
	gridOpts := make([]gridOption, len(options))
	selectedMap := make(map[int]bool)
	for i, opt := range options {
		gridOpts[i] = gridOption{
			value:       opt.Value,
			label:       opt.Label,
			description: opt.Description,
			selected:    preSelectedSet[opt.Value],
		}
		if preSelectedSet[opt.Value] {
			selectedMap[i] = true
		}
	}

	return &gridMultiSelect{
		title:    title,
		options:  gridOpts,
		cursor:   0,
		selected: selectedMap,
		width:    120, // Default, will be updated on window size message
		height:   30,
	}
}

func (m *gridMultiSelect) Init() tea.Cmd {
	return nil
}

func (m *gridMultiSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			m.submitted = true
			return m, tea.Quit

		case " ":
			// Toggle selection
			if m.selected[m.cursor] {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = true
			}

		case "a", "A":
			// Select all
			for i := range m.options {
				m.selected[i] = true
			}

		case "n", "N":
			// Select none
			m.selected = make(map[int]bool)

		case "up", "k":
			m.moveCursor(-m.cols)

		case "down", "j":
			m.moveCursor(m.cols)

		case "left", "h":
			m.moveCursor(-1)

		case "right", "l":
			m.moveCursor(1)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *gridMultiSelect) moveCursor(delta int) {
	newCursor := m.cursor + delta
	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor >= len(m.options) {
		newCursor = len(m.options) - 1
	}
	m.cursor = newCursor
}

func (m *gridMultiSelect) View() string {
	// Calculate grid layout
	itemWidth := 24 // Checkbox + name + padding
	m.cols = m.width / itemWidth
	if m.cols < 1 {
		m.cols = 1
	}
	if m.cols > 5 {
		m.cols = 5 // Max 5 columns for readability
	}

	m.rows = (len(m.options) + m.cols - 1) / m.cols

	// Build the view
	var b strings.Builder

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#89b4fa")). // Blue
		Padding(0, 1)
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Grid
	for row := 0; row < m.rows; row++ {
		for col := 0; col < m.cols; col++ {
			idx := row*m.cols + col
			if idx >= len(m.options) {
				break
			}

			opt := m.options[idx]
			isCursor := idx == m.cursor
			isSelected := m.selected[idx]

			// Build the item
			checkbox := "[ ]"
			if isSelected {
				checkbox = "[x]"
			}

			label := opt.label
			if len(label) > 16 {
				label = label[:13] + "..."
			}

			itemText := fmt.Sprintf("%s %s", checkbox, label)

			// Style based on state
			itemStyle := lipgloss.NewStyle().Width(itemWidth)
			if isCursor {
				itemStyle = itemStyle.
					Foreground(lipgloss.Color("#cba6f7")). // Mauve (cursor)
					Bold(true)
			} else if isSelected {
				itemStyle = itemStyle.
					Foreground(lipgloss.Color("#a6e3a1")) // Green (selected)
			} else {
				itemStyle = itemStyle.
					Foreground(lipgloss.Color("#cdd6f4")) // Text (normal)
			}

			b.WriteString(itemStyle.Render(itemText))
		}
		b.WriteString("\n")
	}

	// Preview pane - show description of current item
	b.WriteString("\n")
	previewStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")). // Subtext
		Italic(true).
		Width(m.width - 4)

	currentOpt := m.options[m.cursor]
	previewText := fmt.Sprintf("Preview: %s - %s", currentOpt.label, currentOpt.description)
	b.WriteString(previewStyle.Render(previewText))

	// Help text
	b.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")) // Overlay0
	helpText := "Navigate: ↑/↓/←/→  Toggle: Space  Select All: A  Select None: N  Continue: Enter  Quit: Esc"
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// runGridMultiSelect runs the grid multi-select component and returns the selected values.
func runGridMultiSelect(title string, options []module.MultiSelectOption, preSelected []string) ([]string, error) {
	m := newGridMultiSelect(title, options, preSelected)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("bubble tea error: %w", err)
	}

	gridModel := finalModel.(*gridMultiSelect)

	if gridModel.cancelled {
		return nil, module.ErrUserCancelled
	}

	// Extract selected values
	var selected []string
	for idx := range gridModel.selected {
		selected = append(selected, gridModel.options[idx].value)
	}

	return selected, nil
}
