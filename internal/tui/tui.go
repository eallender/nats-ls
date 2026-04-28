// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

// Package tui provides terminal UI components for the application
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eallender/nats-ls/internal/config"
	"github.com/eallender/nats-ls/internal/decode"
	"github.com/eallender/nats-ls/internal/logger"
	"github.com/eallender/nats-ls/internal/monitor"
	"github.com/nats-io/nats.go"
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeBrowse ViewMode = iota
	ViewModeLog
	ViewModeMessageInspect
)

// Model represents the application state
type Model struct {
	width    int
	height   int
	quitting bool

	// Connection state
	nc                   *nats.Conn
	serverURL            string
	config               *config.Config
	connectingInProgress bool // Prevents multiple simultaneous connection attempts

	// Command bar state
	commandBarActive      bool
	commandInput          string
	commandPreviousFilter string // Previous filter value before opening command bar

	// Navigation state
	selectedIndex      int
	browseScrollOffset int      // Scroll offset for browse view (top-most visible item)
	navPath            []string // Current navigation path for hierarchical subject browsing
	subjectFilter      string   // Filter pattern for subjects (empty = no filter)
	flatViewMode       bool     // If true, show all subjects in flat list; if false, show hierarchical

	// View mode
	viewMode         ViewMode
	logScrollOffset  int    // Scroll offset for log view
	logSelectedIndex int    // Index of the selected message in log view (relative to visible messages)
	watchingSubject  string // The subject currently being watched in log view

	// Message inspector state
	inspectedMessageIndex int // Index of the message being inspected
	inspectScrollOffset   int // Scroll offset within the inspected message

	// NATS management
	viewer    *monitor.Viewer
	discovery *monitor.Discovery

	// Message decoding
	decoderRegistry *decode.Registry
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
	// Initialize decoder registry
	decoderConfig := decode.DecoderConfig{
		MaxWidth: 80, // Default, will be updated on resize
		Styles: decode.StyleConfig{
			Key:     InspectKeyStyle,
			String:  InspectStringStyle,
			Number:  InspectNumberStyle,
			Bool:    InspectBoolStyle,
			Null:    InspectNullStyle,
			Bracket: InspectBracketStyle,
			Header:  InspectHeaderStyle,
		},
	}

	registry := decode.NewRegistry(&decoderConfig)
	registry.Register(decode.NewJSONDecoder(&decoderConfig))
	registry.Register(decode.NewTextDecoder(&decoderConfig))

	return Model{
		nc:              nc,
		serverURL:       serverURL,
		viewer:          viewer,
		discovery:       discovery,
		config:          cfg,
		decoderRegistry: registry,
	}
}

// Run starts the TUI
func Run(config *config.Config) error {
	var nc *nats.Conn
	var viewer *monitor.Viewer
	var discovery *monitor.Discovery

	var err error
	nc, err = createNATSConnection(config)
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
