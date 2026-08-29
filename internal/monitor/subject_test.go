// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"testing"
	"time"
)

func TestSubjectStoreCleanupRemovesStaleSubjects(t *testing.T) {
	store := NewSubjectStore(10)
	store.Record("stale.subject")
	store.Record("active.subject")

	store.mu.Lock()
	stale, ok := store.subjects["stale.subject"]
	if !ok {
		store.mu.Unlock()
		t.Fatal("expected stale subject to exist")
	}
	stale.LastSeen.Store(time.Now().Add(-10 * time.Minute))
	store.mu.Unlock()

	store.Cleanup(5 * time.Minute)

	subjects := subjectNames(store.All())
	if subjects["stale.subject"] {
		t.Fatal("expected stale subject to be removed")
	}
	if !subjects["active.subject"] {
		t.Fatal("expected active subject to remain")
	}
}

func TestSubjectStoreRecordInitializesLastSeenBeforePublish(t *testing.T) {
	store := NewSubjectStore(10)
	store.Record("test.subject")

	subjects := store.All()
	if len(subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(subjects))
	}
	if _, ok := subjects[0].LastSeen.Load().(time.Time); !ok {
		t.Fatal("expected LastSeen to be initialized")
	}
}

func TestSubjectStoreEnforcesSubjectLimit(t *testing.T) {
	store := NewSubjectStore(2)
	store.Record("subject.1")
	time.Sleep(time.Millisecond)
	store.Record("subject.2")
	time.Sleep(time.Millisecond)
	store.Record("subject.3")

	subjects := subjectNames(store.All())
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got %d", len(subjects))
	}
	if subjects["subject.1"] {
		t.Fatal("expected oldest subject to be pruned")
	}
	if !subjects["subject.2"] || !subjects["subject.3"] {
		t.Fatal("expected newest subjects to remain")
	}
}

func TestSubjectStoreZeroLimitDropsSubjects(t *testing.T) {
	store := NewSubjectStore(0)
	store.Record("test.subject")

	if subjects := store.All(); len(subjects) != 0 {
		t.Fatalf("expected no stored subjects, got %d", len(subjects))
	}
}

func subjectNames(subjects []*SubjectInfo) map[string]bool {
	result := make(map[string]bool, len(subjects))
	for _, subject := range subjects {
		result[subject.Name] = true
	}
	return result
}
