// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	// Navigation state
	selectedIndex int
	navPath       []string // Current navigation path for hierarchical subject browsing

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
