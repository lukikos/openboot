package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/openbootdotdev/openboot/internal/detector"
)

var (
	detSourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888"))

	detInstalledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#22c55e"))

	detMissingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#eab308"))

	detCursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22c55e")).
			Bold(true)

	detHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60a5fa")).
			Bold(true)

	detDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555"))
)

// DetectionModel is a bubbletea Model for selecting detected dependencies.
type DetectionModel struct {
	detections []detector.Detection
	missing    []int          // indices into detections for missing items
	installed  []int          // indices into detections for installed items
	selected   map[int]bool   // which missing indices are selected
	cursor     int            // cursor position within missing items
	confirmed  bool
	cancelled  bool
	width      int
	height     int
}

func newDetectionModel(detections []detector.Detection) DetectionModel {
	var missing, installed []int
	selected := make(map[int]bool)

	for i, d := range detections {
		if d.Installed {
			installed = append(installed, i)
		} else {
			missing = append(missing, i)
			// Pre-select Required and Recommended
			if d.Confidence != detector.ConfidenceOptional {
				selected[i] = true
			}
		}
	}

	return DetectionModel{
		detections: detections,
		missing:    missing,
		installed:  installed,
		selected:   selected,
	}
}

func (m DetectionModel) Init() tea.Cmd {
	return nil
}

func (m DetectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "esc"))):
			m.cancelled = true
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			m.confirmed = true
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.missing)-1 {
				m.cursor++
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
			if m.cursor < len(m.missing) {
				idx := m.missing[m.cursor]
				m.selected[idx] = !m.selected[idx]
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			// Toggle all
			allSelected := true
			for _, idx := range m.missing {
				if !m.selected[idx] {
					allSelected = false
					break
				}
			}
			for _, idx := range m.missing {
				m.selected[idx] = !allSelected
			}
		}
	}

	return m, nil
}

func (m DetectionModel) View() string {
	var b strings.Builder

	// Missing section
	if len(m.missing) > 0 {
		b.WriteString(detHeaderStyle.Render("  MISSING — select packages to install:"))
		b.WriteString("\n")

		for i, idx := range m.missing {
			d := m.detections[idx]

			// Cursor
			cursor := "  "
			if i == m.cursor {
				cursor = detCursorStyle.Render("> ")
			}

			// Checkbox
			checkbox := "[ ]"
			if m.selected[idx] {
				checkbox = detInstalledStyle.Render("[x]")
			}

			// Package name with version
			name := d.Package
			if d.Version != "" {
				name += "@" + d.Version
			}

			// Confidence hint for optional
			source := d.Source
			if d.Confidence == detector.ConfidenceOptional {
				source += " (optional)"
			}

			line := fmt.Sprintf("%s %s %-18s %s  %s",
				cursor,
				checkbox,
				name,
				detDimStyle.Render(d.Description),
				detSourceStyle.Render("from "+source),
			)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Installed section
	if len(m.installed) > 0 {
		b.WriteString("\n")
		b.WriteString(detHeaderStyle.Render("  INSTALLED:"))
		b.WriteString("\n")

		for _, idx := range m.installed {
			d := m.detections[idx]
			name := d.Package
			if d.Version != "" {
				name += "@" + d.Version
			}

			line := fmt.Sprintf("    %s %-18s %s  %s",
				detInstalledStyle.Render("✓"),
				name,
				detDimStyle.Render(d.Description),
				detSourceStyle.Render("from "+d.Source),
			)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Footer
	selectedCount := 0
	for _, idx := range m.missing {
		if m.selected[idx] {
			selectedCount++
		}
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %d of %d selected", selectedCount, len(m.missing)))
	b.WriteString(detDimStyle.Render("  •  space: toggle  enter: install  a: all  q: quit"))
	b.WriteString("\n")

	return b.String()
}

// RunDetectionSelector runs the interactive detection TUI and returns selected packages.
// Returns nil if the user cancelled.
func RunDetectionSelector(detections []detector.Detection) ([]detector.Detection, error) {
	model := newDetectionModel(detections)

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("run TUI: %w", err)
	}

	m, ok := finalModel.(DetectionModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type from TUI")
	}
	if m.cancelled || !m.confirmed {
		return nil, nil
	}

	var selected []detector.Detection
	for _, idx := range m.missing {
		if m.selected[idx] {
			selected = append(selected, m.detections[idx])
		}
	}

	return selected, nil
}
