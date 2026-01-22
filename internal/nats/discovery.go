// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package nats

import (
	"context"

	"github.com/nats-io/nats.go"
)

type Discovery struct {
	nc    *nats.Conn
	sub   *nats.Subscription
	store *SubjectStore
}

func NewDiscovery(nc *nats.Conn, store *SubjectStore) *Discovery {
	return &Discovery{
		nc:    nc,
		store: store,
	}
}

func (d *Discovery) Start(ctx context.Context, maxMessages int, maxStorageMB int) error {
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
		d.sub.Unsubscribe()
	}()

	return nil
}
