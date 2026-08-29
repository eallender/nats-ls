// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

// Package monitor implements the core nats message ingestion logic
package monitor

import (
	"sync"

	"github.com/eallender/nats-ls/internal/logger"
	"github.com/nats-io/nats.go"
)

type Viewer struct {
	nc            *nats.Conn
	sub           *nats.Subscription
	mu            sync.Mutex
	messages      *MessageStore
	paused        bool
	pausedSubject string // Subject to resubscribe to when resuming
}

func NewViewer(nc *nats.Conn, maxMessages int) *Viewer {
	return &Viewer{
		nc:       nc,
		messages: NewMessageStore(maxMessages),
	}
}

// Watch points the Viewer to a new NATS subject
func (v *Viewer) Watch(subject string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.messages.Count() != 0 {
		v.messages.Clear()
	}

	if v.sub != nil {
		if err := v.sub.Unsubscribe(); err != nil {
			logger.Log.Warn("Failed to unsubscribe during watch change", "error", err)
		}
		v.sub = nil
	}

	v.paused = false
	v.pausedSubject = subject

	if subject == "" {
		return nil
	}

	var err error
	v.sub, err = v.nc.Subscribe(subject, func(msg *nats.Msg) {
		v.messages.Store(msg)
	})
	if err != nil {
		return err
	}
	logger.Log.Info("Subscribed to subject", "subject", subject)

	return err
}

// Stop the Viewer from ingesting NATS messages
func (v *Viewer) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sub != nil {
		if err := v.sub.Unsubscribe(); err != nil {
			logger.Log.Warn("Failed to unsubscribe during viewer stop", "error", err)
		}
		v.sub = nil
	}
	if v.messages.Count() != 0 {
		v.messages.Clear()
	}
	logger.Log.Debug("Viewer has been stopped")
}

// GetMessages returns all stored messages
func (v *Viewer) GetMessages() []Message {
	return v.messages.All()
}

// GetMessageCount returns the number of stored messages
func (v *Viewer) GetMessageCount() int {
	return v.messages.Count()
}

// Pause temporarily stops receiving new messages without clearing the buffer
func (v *Viewer) Pause() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.paused || v.sub == nil {
		return
	}

	v.paused = true
	if err := v.sub.Unsubscribe(); err != nil {
		logger.Log.Warn("Failed to unsubscribe during pause", "error", err)
	}
	v.sub = nil
	logger.Log.Debug("Viewer paused")
}

// Resume restarts receiving messages after a pause
func (v *Viewer) Resume() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.paused || v.pausedSubject == "" {
		return nil
	}

	var err error
	v.sub, err = v.nc.Subscribe(v.pausedSubject, func(msg *nats.Msg) {
		v.messages.Store(msg)
	})
	if err != nil {
		return err
	}

	v.paused = false
	logger.Log.Debug("Viewer resumed")
	return nil
}

// IsPaused returns whether the viewer is currently paused
func (v *Viewer) IsPaused() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.paused
}
