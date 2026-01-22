// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package nats

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

func (s *SubjectStore) All() []*SubjectInfo {
	var result []*SubjectInfo
	s.subjects.Range(func(_, value any) bool {
		result = append(result, value.(*SubjectInfo))
		return true
	})
	return result
}

func (s *SubjectStore) Get(subject string) (*SubjectInfo, bool) {
	val, ok := s.subjects.Load(subject)
	if !ok {
		return nil, false
	}
	return val.(*SubjectInfo), true
}
