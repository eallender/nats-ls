// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"sync"
	"time"

	"github.com/eallender/nats-ls/internal/logger"
	"github.com/nats-io/nats.go"
)

const (
	DefaultSubjectCleanupInterval = 1 * time.Minute
	DefaultSubjectMaxAge          = 5 * time.Minute
)

type Discovery struct {
	nc              *nats.Conn
	sub             *nats.Subscription
	mu              sync.Mutex
	store           *SubjectStore
	done            chan struct{}
	cleanupInterval time.Duration
	subjectMaxAge   time.Duration
}

func NewDiscovery(nc *nats.Conn, maxSubjects int, subjectMaxAge time.Duration, cleanupInterval time.Duration) *Discovery {
	if subjectMaxAge <= 0 {
		subjectMaxAge = DefaultSubjectMaxAge
	}
	if cleanupInterval <= 0 {
		cleanupInterval = DefaultSubjectCleanupInterval
	}

	return &Discovery{
		nc:              nc,
		store:           NewSubjectStore(maxSubjects),
		cleanupInterval: cleanupInterval,
		subjectMaxAge:   subjectMaxAge,
	}
}

// Start NATS subject discovery
func (d *Discovery) Start(maxMessages int, maxStorageMB int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.sub != nil {
		return nil
	}

	var err error
	d.sub, err = d.nc.Subscribe(">", func(msg *nats.Msg) {
		d.store.Record(msg.Subject)
	})
	if err != nil {
		return err
	}

	if err := d.sub.SetPendingLimits(maxMessages, maxStorageMB*1024*1024); err != nil {
		if unsubErr := d.sub.Unsubscribe(); unsubErr != nil {
			logger.Log.Warn("Failed to unsubscribe after setting discovery pending limits failed", "error", unsubErr)
		}
		d.sub = nil
		return err
	}

	d.done = make(chan struct{})
	go d.cleanupLoop(d.done)

	return nil
}

func (d *Discovery) cleanupLoop(done <-chan struct{}) {
	ticker := time.NewTicker(d.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			d.store.Cleanup(d.subjectMaxAge)
		}
	}
}

// GetAllSubjects returns all discovered subjects
func (d *Discovery) GetAllSubjects() []*SubjectInfo {
	return d.store.All()
}

// Stop unsubscribes and cleans up the discovery
func (d *Discovery) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.done != nil {
		close(d.done)
		d.done = nil
	}
	if d.sub != nil {
		if err := d.sub.Unsubscribe(); err != nil {
			logger.Log.Warn("Failed to unsubscribe during discovery stop", "error", err)
		}
		d.sub = nil
	}
	logger.Log.Debug("Discovery has been stopped")
}
