// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package nats

import (
	"fmt"
	"io"
	"sync"

	"github.com/eallender/nats-ls/internal/logger"
	"github.com/nats-io/nats.go"
)

type Viewer struct {
	nc     *nats.Conn
	sub    *nats.Subscription
	mu     sync.Mutex
	output io.Writer
}

func NewViewer(nc *nats.Conn, output io.Writer) *Viewer {
	return &Viewer{
		nc:     nc,
		output: output,
	}
}

func (v *Viewer) Watch(subject string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sub != nil {
		v.sub.Unsubscribe()
		v.sub = nil
	}

	if subject == "" {
		return nil
	}

	var err error
	v.sub, err = v.nc.Subscribe(subject, func(msg *nats.Msg) {
		fmt.Fprintf(v.output, "[%s] %s\n", msg.Subject, string(msg.Data))
	})
	logger.Log.Debug("Viewer subscribed to", "subject", subject)

	return err
}

func (v *Viewer) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sub != nil {
		v.sub.Unsubscribe()
		v.sub = nil
	}
	logger.Log.Debug("Viewer has been stopped")
}
