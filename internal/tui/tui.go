// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eallender/nats-ls/internal/config"
	"github.com/eallender/nats-ls/internal/logger"
	"github.com/eallender/nats-ls/internal/monitor"
	"github.com/nats-io/nats.go"
)

// Model represents the application state
type Model struct {
	width    int
	height   int
	quitting bool

	// Connection state
	nc           *nats.Conn
	serverURL    string
	messageCount int
	config       *config.Config

	// Command bar state
	commandBarActive bool
	commandInput     string

	// NATS management
	viewer    *monitor.Viewer
	discovery *monitor.Discovery
}

// connectAttemptMsg is sent when a connection attempt completes
type connectAttemptMsg struct {
	nc        *nats.Conn
	viewer    *monitor.Viewer
	discovery *monitor.Discovery
	err       error
}

// tickMsg is sent periodically to refresh the UI and retry connections
type tickMsg time.Time

// New creates a new TUI model
func New(nc *nats.Conn, viewer *monitor.Viewer, discovery *monitor.Discovery, serverURL string, cfg *config.Config) Model {
	return Model{
		nc:           nc,
		serverURL:    serverURL,
		messageCount: 0,
		viewer:       viewer,
		discovery:    discovery,
		config:       cfg,
	}
}

// tryConnect attempts to connect to NATS and returns a command
func (m Model) tryConnect() tea.Msg {
	nc, err := nats.Connect(
		m.config.NatsAddress,
		nats.MaxReconnects(m.config.NatsMaxReconnects),
		nats.ReconnectWait(time.Duration(m.config.NatsReconnectWaitSeconds)*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Log.Warn("Disconnected from NATS", "error", err)
			} else {
				logger.Log.Info("Disconnected from NATS")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Log.Info("Reconnected to NATS", "address", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Log.Debug("NATS connection closed")
		}),
	)

	if err != nil {
		logger.Log.Debug("Connection attempt failed", "error", err)
		return connectAttemptMsg{nc: nil, err: err}
	}

	logger.Log.Info("Connected to NATS", "address", m.config.NatsAddress)
	viewer := monitor.NewViewer(nc, m.config.NatsViewerMessageLimit)
	discovery := monitor.NewDiscovery(nc)

	// Start discovery to listen for all subjects
	ctx := context.Background()
	if err := discovery.Start(ctx, m.config.NatsDiscoveryPendingLimit, m.config.NatsDiscoveryStorageLimitMB); err != nil {
		logger.Log.Warn("Failed to start discovery", "error", err)
	}

	return connectAttemptMsg{
		nc:        nc,
		viewer:    viewer,
		discovery: discovery,
		err:       nil,
	}
}

// tickCmd sends a tick message after a delay to refresh the UI and retry connections
func tickCmd() tea.Msg {
	time.Sleep(1 * time.Second)
	return tickMsg(time.Now())
}

// IsConnected checks if we're connected to NATS
func (m Model) IsConnected() bool {
	return m.nc != nil && m.nc.IsConnected()
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	// If not connected, start trying to connect
	if !m.IsConnected() {
		return m.tryConnect
	}
	// Start the tick loop to refresh the UI
	return tickCmd
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
	case connectAttemptMsg:
		if msg.err != nil {
			// Connection failed, retry after a delay
			return m, tickCmd
		}
		// Connection successful, update model
		m.nc = msg.nc
		m.viewer = msg.viewer
		m.discovery = msg.discovery
		// Start the tick loop to refresh the UI
		return m, tickCmd
	case tickMsg:
		// If not connected, try to reconnect
		if !m.IsConnected() {
			return m, tea.Batch(m.tryConnect, tickCmd)
		}
		// Otherwise just refresh the UI periodically to show new subjects
		return m, tickCmd
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
	if m.IsConnected() {
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

	// Build navigation content with discovered subjects
	var navText string
	if m.discovery != nil {
		subjects := m.discovery.GetAllSubjects()
		if len(subjects) > 0 {
			navText = "Discovered Subjects:\n\n"
			for _, subject := range subjects {
				navText += fmt.Sprintf("• %s (%d)\n", subject.Name, subject.MessageCount.Load())
			}
		} else {
			navText = "Discovered Subjects:\n\nNo subjects discovered yet..."
		}
	} else {
		navText = "Discovered Subjects:\n\nNot connected..."
	}

	// Navigation panel (1/3 width)
	// Subtract padding (4) and borders (2)
	navContent := NavStyle.
		Width(navWidth).
		Height(m.height - 10).
		Render(navText)

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
func Run(config *config.Config) error {
	var nc *nats.Conn
	var viewer *monitor.Viewer
	var discovery *monitor.Discovery

	var err error
	nc, err = nats.Connect(
		config.NatsAddress,
		nats.MaxReconnects(config.NatsMaxReconnects),
		nats.ReconnectWait(time.Duration(config.NatsReconnectWaitSeconds)*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Log.Warn("Disconnected from NATS", "error", err)
			} else {
				logger.Log.Info("Disconnected from NATS")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Log.Info("Reconnected to NATS", "address", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Log.Debug("NATS connection closed")
		}),
	)
	if err != nil {
		// Initial connection failed, but continue with TUI
		logger.Log.Warn("Could not connect to NATS", "address", config.NatsAddress, "error", err)
	} else {
		viewer = monitor.NewViewer(nc, config.NatsViewerMessageLimit)
		discovery = monitor.NewDiscovery(nc)

		// Start discovery to listen for all subjects
		ctx := context.Background()
		if err := discovery.Start(ctx, config.NatsDiscoveryPendingLimit, config.NatsDiscoveryStorageLimitMB); err != nil {
			logger.Log.Warn("Failed to start discovery", "error", err)
		}

		logger.Log.Info("Connected to NATS", "address", config.NatsAddress)
	}

	p := tea.NewProgram(New(nc, viewer, discovery, config.NatsAddress, config), tea.WithAltScreen())
	finalModel, err := p.Run()

	// Clean up connections from the final model state
	if m, ok := finalModel.(Model); ok {
		if m.viewer != nil {
			m.viewer.Stop()
		}
		if m.discovery != nil {
			m.discovery.Stop()
		}
		if m.nc != nil && m.nc.IsConnected() {
			m.nc.Close()
		}
	}

	return err
}
