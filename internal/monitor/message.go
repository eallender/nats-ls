// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type Message struct {
	Subject   string
	Data      []byte
	Timestamp time.Time
	Headers   nats.Header
}

type MessageStore struct {
	mu       sync.RWMutex
	messages []Message
	maxSize  int
	next     int
	count    int
}

// NewMessageStore creates a new Message Store
func NewMessageStore(maxSize int) *MessageStore {
	if maxSize < 0 {
		maxSize = 0
	}

	return &MessageStore{
		messages: make([]Message, maxSize),
		maxSize:  maxSize,
	}
}

// Store adds a message to the store, overwriting the oldest message if at capacity.
func (m *MessageStore) Store(natsMsg *nats.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.maxSize == 0 {
		return
	}

	m.messages[m.next] = Message{
		Subject:   natsMsg.Subject,
		Data:      natsMsg.Data,
		Timestamp: time.Now(),
		Headers:   natsMsg.Header,
	}

	m.next = (m.next + 1) % m.maxSize
	if m.count < m.maxSize {
		m.count++
	}
}

// Clear removes all messages from the store
func (m *MessageStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	clear(m.messages)
	m.next = 0
	m.count = 0
}

// All returns a copy of all messages in oldest-to-newest order.
func (m *MessageStore) All() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, m.count)
	if m.count == 0 {
		return result
	}

	start := 0
	if m.count == m.maxSize {
		start = m.next
	}

	for i := range m.count {
		result[i] = m.messages[(start+i)%m.maxSize]
	}

	return result
}

// Count returns the number of messages currently stored
func (m *MessageStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.count
}
