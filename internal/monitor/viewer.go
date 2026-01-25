// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"sync"

	"github.com/eallender/nats-ls/internal/logger"
	"github.com/nats-io/nats.go"
)

type Viewer struct {
	nc       *nats.Conn
	sub      *nats.Subscription
	mu       sync.Mutex
	messages *MessageStore
}

func NewViewer(nc *nats.Conn, maxMessages int) *Viewer {
	return &Viewer{
		nc:       nc,
		messages: NewMessageStore(maxMessages),
	}
}

// Points the Viewer to a new NATS subject
func (v *Viewer) Watch(subject string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.messages.Count() != 0 {
		v.messages.Clear()
	}

	if v.sub != nil {
		v.sub.Unsubscribe()
		v.sub = nil
	}

	if subject == "" {
		return nil
	}

	var err error
	v.sub, err = v.nc.Subscribe(subject, func(msg *nats.Msg) {
		v.messages.Store(msg)
		logger.Log.Debug("Message received", "subject", msg.Subject, "size", len(msg.Data))
	})
	if err != nil {
		return err
	}
	logger.Log.Info("Subscribed to subject", "subject", subject)

	return err
}

// Stops the Viewer from ingesting NATS messages
func (v *Viewer) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sub != nil {
		v.sub.Unsubscribe()
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
