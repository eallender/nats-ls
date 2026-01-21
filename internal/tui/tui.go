// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type Model struct {
	width    int
	height   int
	quitting bool

	// Connection state
	connected    bool
	serverURL    string
	messageCount int

	// Command bar state
	commandBarActive bool
	commandInput     string

	// Add NATS-specific state here:
	// subjects []string
	// messages []NatsMessage
}

// New creates a new TUI model
func New() Model {
	return Model{
		connected:    false,
		serverURL:    "nats://localhost:4222", // default, will be configurable
		messageCount: 0,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return nil // or return a command to connect to NATS
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If command bar is active, handle its input
		if m.commandBarActive {
			switch msg.String() {
			case "enter":
				// TODO: Process command in m.commandInput
				m.commandBarActive = false
				m.commandInput = ""
			case "esc":
				m.commandBarActive = false
				m.commandInput = ""
			case "backspace":
				if len(m.commandInput) > 0 {
					m.commandInput = m.commandInput[:len(m.commandInput)-1]
				}
			default:
				// Add character to input
				m.commandInput += msg.String()
			}
			return m, nil
		}

		// Normal mode key handling
		switch msg.String() {
		case ":":
			m.commandBarActive = true
			m.commandInput = ""
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

// renderHeader creates the header bar with app info and status
func (m Model) renderHeader() string {
	// ASCII art logo
	logo := HeaderAppNameStyle.Render(Logo)

	// Connection status
	var statusText string
	var statusStyle lipgloss.Style
	if m.connected {
		statusStyle = HeaderConnectedStyle
		statusText = "● Connected"
	} else {
		statusStyle = HeaderDisconnectedStyle
		statusText = "● Disconnected"
	}

	status := statusStyle.Render(statusText)
	server := HeaderServerStyle.Render(fmt.Sprintf("Server: %s", m.serverURL))
	msgCount := HeaderStatsStyle.Render(fmt.Sprintf("Messages: %d", m.messageCount))
	statusInfo := HeaderStatusInfoStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		status,
		server,
		msgCount,
	))

	controls := HeaderControlStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"<enter>",
		"<↑↓>",
		"<l>",
		"<:>",
		"<q>",
	),
	)

	controlsInfo := HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"inspect",
		"navigate",
		"logs",
		"filter",
		"quit",
	))

	// Combine logo and status horizontally
	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logo,
		statusInfo,
		controls,
		controlsInfo,
	)

	// Apply container style with padding and width
	return HeaderContainerStyle.
		Width(m.width).
		Padding(0, 1).
		Render(headerContent)
}

// renderContent creates the main content area with nav and info panels
func (m Model) renderContent() string {
	// Calculate widths: Nav = 1/3, Info = fills remaining space
	navWidth := m.width / 3
	infoWidth := m.width - navWidth

	// Navigation panel (1/3 width)
	// Subtract padding (4) and borders (2)
	navContent := NavStyle.
		Width(navWidth).
		Height(m.height - 10).
		Render("Navigation\n\n• Item 1\n• Item 2\n• Item 3")

	// Info/main content panel (fills remaining space)
	// Subtract padding (4) and borders (2)
	infoContent := InfoStyle.
		Width(infoWidth - 6).
		Height(m.height - 10).
		Render("Info Panel\n\nContent goes here")

	// Combine navigation and info horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		navContent,
		infoContent,
	)

	return content
}

// renderCommandBar creates the command input bar
func (m Model) renderCommandBar() string {
	if !m.commandBarActive {
		return ""
	}

	prompt := CommandBarStyle.
		Width(m.width).
		Render(fmt.Sprintf(":%s", m.commandInput))
	return prompt
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Build the complete UI
	header := m.renderHeader()
	commandBar := m.renderCommandBar()
	content := m.renderContent()

	// Combine all sections
	if m.commandBarActive {
		return lipgloss.JoinVertical(lipgloss.Left, header, commandBar, content)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

// Run starts the TUI
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
