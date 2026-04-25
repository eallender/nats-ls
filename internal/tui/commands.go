// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eallender/nats-ls/internal/config"
	"github.com/eallender/nats-ls/internal/logger"
	"github.com/eallender/nats-ls/internal/monitor"
	"github.com/nats-io/nats.go"
)

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	if !m.IsConnected() {
		return m.tryConnect()
	}
	// Start the tick loop to refresh UI
	return tickCmd()
}

// createNATSConnection creates a NATS connection with standard handlers
func createNATSConnection(cfg *config.Config) (*nats.Conn, error) {
	return nats.Connect(
		cfg.NatsAddress,
		nats.Timeout(3*time.Second), // Timeout for initial connection attempt
		nats.MaxReconnects(cfg.NatsMaxReconnects),
		nats.ReconnectWait(time.Duration(cfg.NatsReconnectWaitSeconds)*time.Second),
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
}

// tryConnect attempts to connect to NATS asynchronously and returns a command
func (m Model) tryConnect() tea.Cmd {
	return func() tea.Msg {
		nc, err := createNATSConnection(m.config)
		if err != nil {
			logger.Log.Debug("Connection attempt failed", "error", err)
			return connectAttemptMsg{nc: nil, err: err}
		}

		logger.Log.Info("Connected to NATS", "address", m.config.NatsAddress)
		viewer := monitor.NewViewer(nc, m.config.NatsViewerMessageLimit)
		discovery := monitor.NewDiscovery(
			nc,
			m.config.NatsDiscoverySubjectLimit,
			time.Duration(m.config.NatsDiscoverySubjectMaxAgeMinutes)*time.Minute,
			time.Duration(m.config.NatsDiscoveryCleanupIntervalSeconds)*time.Second,
		)

		// Start discovery to listen for all subjects
		if err := discovery.Start(m.config.NatsDiscoveryPendingLimit, m.config.NatsDiscoveryStorageLimitMB); err != nil {
			logger.Log.Warn("Failed to start discovery", "error", err)
		}

		return connectAttemptMsg{
			nc:        nc,
			viewer:    viewer,
			discovery: discovery,
			err:       nil,
		}
	}
}

// tickCmd returns a command that sends a tick message after a delay to refresh the UI and retry connections
func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// tickCmdSlow returns a command with a longer delay for connection retries
func tickCmdSlow() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// IsConnected checks if we're connected to NATS
func (m Model) IsConnected() bool {
	return m.nc != nil && m.nc.IsConnected()
}

func (m Model) shouldAttemptConnection() bool {
	return m.nc == nil || m.nc.IsClosed()
}
