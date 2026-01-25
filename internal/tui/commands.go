// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eallender/nats-ls/internal/logger"
	"github.com/eallender/nats-ls/internal/monitor"
	"github.com/nats-io/nats.go"
)

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	// If not connected, start trying to connect
	if !m.IsConnected() {
		return m.tryConnect
	}
	// Start the tick loop to refresh the UI
	return tickCmd
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
