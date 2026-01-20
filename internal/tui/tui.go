// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type Model struct {
	width    int
	height   int
	quitting bool
	// Add NATS-specific state here:
	// subjects []string
	// messages []NatsMessage
}

// New creates a new TUI model
func New() Model {
	return Model{}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil // or return a command to connect to NATS
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	return style.Render("nats-ls - Press 'q' to quit\n")
}

// Run starts the TUI
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
