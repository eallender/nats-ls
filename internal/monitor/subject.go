// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package monitor

import (
	"sync"
	"sync/atomic"
	"time"
)

type SubjectInfo struct {
	Name         string
	FirstSeen    time.Time
	LastSeen     atomic.Value
	MessageCount atomic.Int64
}

type SubjectStore struct {
	mu          sync.RWMutex
	subjects    map[string]*SubjectInfo
	maxSubjects int
}

func NewSubjectStore(maxSubjects int) *SubjectStore {
	if maxSubjects < 0 {
		maxSubjects = 0
	}

	return &SubjectStore{
		subjects:    make(map[string]*SubjectInfo),
		maxSubjects: maxSubjects,
	}
}

// Record an encountered subject in the subject store
func (s *SubjectStore) Record(subject string) (isNew bool) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxSubjects == 0 {
		return true
	}

	info, ok := s.subjects[subject]
	if !ok {
		info = &SubjectInfo{
			Name:      subject,
			FirstSeen: now,
		}
		info.LastSeen.Store(now)
		s.subjects[subject] = info
		s.pruneToLimitLocked()
	} else {
		info.LastSeen.Store(now)
	}

	info.MessageCount.Add(1)
	return !ok
}

// All converts the subject store map to a usable slice
func (s *SubjectStore) All() []*SubjectInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*SubjectInfo, 0, len(s.subjects))
	for _, value := range s.subjects {
		result = append(result, value)
	}
	return result
}

// Cleanup removes subjects not seen within maxAge and enforces the subject limit.
func (s *SubjectStore) Cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxAge > 0 {
		cutoff := time.Now().Add(-maxAge)
		for subject, info := range s.subjects {
			lastSeen, ok := info.LastSeen.Load().(time.Time)
			if ok && lastSeen.Before(cutoff) {
				delete(s.subjects, subject)
			}
		}
	}

	s.pruneToLimitLocked()
}

func (s *SubjectStore) pruneToLimitLocked() {
	if s.maxSubjects <= 0 {
		return
	}

	for len(s.subjects) > s.maxSubjects {
		oldestSubject := ""
		oldestSeen := time.Time{}

		for subject, info := range s.subjects {
			lastSeen, ok := info.LastSeen.Load().(time.Time)
			if !ok {
				lastSeen = info.FirstSeen
			}
			if oldestSubject == "" || lastSeen.Before(oldestSeen) {
				oldestSubject = subject
				oldestSeen = lastSeen
			}
		}

		if oldestSubject == "" {
			return
		}
		delete(s.subjects, oldestSubject)
	}
}
