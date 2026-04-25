// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestMessageStoreKeepsOnlyNewestMessages(t *testing.T) {
	store := NewMessageStore(3)

	for i := range 5 {
		store.Store(&nats.Msg{Subject: "test", Data: []byte(fmt.Sprintf("msg-%d", i))})
	}

	messages := store.All()
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	for i, message := range messages {
		expected := fmt.Sprintf("msg-%d", i+2)
		if string(message.Data) != expected {
			t.Fatalf("expected message %d to be %q, got %q", i, expected, string(message.Data))
		}
	}
}

func TestMessageStoreReturnsOldestToNewestBeforeCapacity(t *testing.T) {
	store := NewMessageStore(3)

	for i := range 2 {
		store.Store(&nats.Msg{Subject: "test", Data: []byte(fmt.Sprintf("msg-%d", i))})
	}

	messages := store.All()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	for i, message := range messages {
		expected := fmt.Sprintf("msg-%d", i)
		if string(message.Data) != expected {
			t.Fatalf("expected message %d to be %q, got %q", i, expected, string(message.Data))
		}
	}
}

func TestMessageStoreZeroCapacityDropsMessages(t *testing.T) {
	store := NewMessageStore(0)
	store.Store(&nats.Msg{Subject: "test", Data: []byte("msg")})

	if count := store.Count(); count != 0 {
		t.Fatalf("expected 0 messages, got %d", count)
	}
}

func TestMessageStoreNegativeCapacityDropsMessages(t *testing.T) {
	store := NewMessageStore(-1)
	store.Store(&nats.Msg{Subject: "test", Data: []byte("msg")})

	if count := store.Count(); count != 0 {
		t.Fatalf("expected 0 messages, got %d", count)
	}
}

func TestMessageStoreClearReleasesReferencesAndResetsRing(t *testing.T) {
	store := NewMessageStore(2)
	store.Store(&nats.Msg{Subject: "test", Data: []byte("msg-0")})
	store.Store(&nats.Msg{Subject: "test", Data: []byte("msg-1")})
	store.Store(&nats.Msg{Subject: "test", Data: []byte("msg-2")})
	store.Clear()

	if count := store.Count(); count != 0 {
		t.Fatalf("expected 0 messages after clear, got %d", count)
	}

	store.Store(&nats.Msg{Subject: "test", Data: []byte("msg-3")})
	messages := store.All()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message after reusing cleared store, got %d", len(messages))
	}
	if string(messages[0].Data) != "msg-3" {
		t.Fatalf("expected msg-3 after reusing cleared store, got %q", string(messages[0].Data))
	}
}
