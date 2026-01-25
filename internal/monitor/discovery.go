// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"context"
	"sync"

	"github.com/eallender/nats-ls/internal/logger"
	"github.com/nats-io/nats.go"
)

type Discovery struct {
	nc    *nats.Conn
	sub   *nats.Subscription
	mu    sync.Mutex
	store *SubjectStore
}

func NewDiscovery(nc *nats.Conn) *Discovery {
	return &Discovery{
		nc:    nc,
		store: &SubjectStore{},
	}
}

// Starts NATS subject discovery
func (d *Discovery) Start(ctx context.Context, maxMessages int, maxStorageMB int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var err error
	d.sub, err = d.nc.Subscribe(">", func(msg *nats.Msg) {
		d.store.Record(msg.Subject)
	})
	if err != nil {
		return err
	}

	d.sub.SetPendingLimits(maxMessages, maxStorageMB*1024*1024)

	go func() {
		<-ctx.Done()
		d.Stop()
	}()

	return nil
}

// GetAllSubjects returns all discovered subjects
func (d *Discovery) GetAllSubjects() []*SubjectInfo {
	return d.store.All()
}

// GetSubject returns info for a specific subject
func (d *Discovery) GetSubject(subject string) (*SubjectInfo, bool) {
	return d.store.Get(subject)
}

// Stop unsubscribes and cleans up the discovery
func (d *Discovery) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.sub != nil {
		d.sub.Unsubscribe()
		d.sub = nil
	}
	logger.Log.Debug("Discovery has been stopped")
}
