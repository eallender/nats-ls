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
}

// Creates a new Message Store
func NewMessageStore(maxSize int) *MessageStore {
	return &MessageStore{
		messages: make([]Message, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Store adds a message to the store, removing oldest if at capacity
func (m *MessageStore) Store(natsMsg *nats.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert nats.Msg to our Message struct
	message := Message{
		Subject:   natsMsg.Subject,
		Data:      natsMsg.Data,
		Timestamp: time.Now(),
		Headers:   natsMsg.Header,
	}

	// If at capacity, remove oldest (shift left)
	if len(m.messages) >= m.maxSize {
		m.messages = m.messages[1:]
	}

	m.messages = append(m.messages, message)
}

// Clear removes all messages from the store
func (m *MessageStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]Message, 0, m.maxSize)
}

// All returns a copy of all messages
func (m *MessageStore) All() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// Count returns the number of messages currently stored
func (m *MessageStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.messages)
}
