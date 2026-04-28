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
	subjects sync.Map
}

// Record an encountered subject in the subject store
func (s *SubjectStore) Record(subject string) (isNew bool) {
	now := time.Now()

	actual, loaded := s.subjects.LoadOrStore(subject, &SubjectInfo{
		Name:      subject,
		FirstSeen: now,
	})

	info := actual.(*SubjectInfo)
	info.LastSeen.Store(now)
	info.MessageCount.Add(1)

	return !loaded
}

// All converts the subject store map to a usable slice
func (s *SubjectStore) All() []*SubjectInfo {
	var result []*SubjectInfo
	s.subjects.Range(func(_, value any) bool {
		result = append(result, value.(*SubjectInfo))
		return true
	})
	return result
}
