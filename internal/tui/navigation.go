// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"sort"
	"strings"
	"time"
)

// SubjectNode represents a subject or subject prefix in the hierarchy
type SubjectNode struct {
	Name         string
	IsLeaf       bool // true if this is a complete subject, false if it's a prefix
	MessageCount int64
	FirstSeen    time.Time
	LastSeen     time.Time
}

// getSubjectsAtCurrentLevel returns the subjects/prefixes at the current navigation level
func (m Model) getSubjectsAtCurrentLevel() []SubjectNode {
	if m.discovery == nil {
		return nil
	}

	subjects := m.discovery.GetAllSubjects()
	if len(subjects) == 0 {
		return nil
	}

	// In flat view mode, return all subjects as leaf nodes (no hierarchy)
	if m.flatViewMode {
		var nodes []SubjectNode
		for _, subject := range subjects {
			// Apply subject filter if active
			if m.subjectFilter != "" {
				// Case-insensitive substring match
				if !strings.Contains(strings.ToLower(subject.Name), strings.ToLower(m.subjectFilter)) {
					continue
				}
			}

			lastSeen := subject.LastSeen.Load().(time.Time)
			nodes = append(nodes, SubjectNode{
				Name:         subject.Name,
				IsLeaf:       true, // All subjects are leaves in flat mode
				MessageCount: subject.MessageCount.Load(),
				FirstSeen:    subject.FirstSeen,
				LastSeen:     lastSeen,
			})
		}

		// Sort alphabetically to maintain consistent order
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})

		return nodes
	}

	// Hierarchical view mode - original behavior
	// Build the current prefix from navPath
	currentPrefix := strings.Join(m.navPath, ".")
	if currentPrefix != "" {
		currentPrefix += "."
	}

	// Group subjects by the next level
	nodeMap := make(map[string]*SubjectNode)

	for _, subject := range subjects {
		// Skip subjects that don't match our current prefix
		if currentPrefix != "" && !strings.HasPrefix(subject.Name, currentPrefix) {
			continue
		}

		// Apply subject filter if active
		if m.subjectFilter != "" {
			// Case-insensitive substring match
			if !strings.Contains(strings.ToLower(subject.Name), strings.ToLower(m.subjectFilter)) {
				continue
			}
		}

		// Get the part after the current prefix
		remainder := strings.TrimPrefix(subject.Name, currentPrefix)

		// Split by "." to get the next level
		parts := strings.Split(remainder, ".")

		if len(parts) > 0 && parts[0] != "" {
			nextLevel := parts[0]
			isLeaf := len(parts) == 1

			lastSeen := subject.LastSeen.Load().(time.Time)

			if existing, ok := nodeMap[nextLevel]; ok {
				existing.MessageCount += subject.MessageCount.Load()

				if isLeaf {
					existing.IsLeaf = true
				}

				if subject.FirstSeen.Before(existing.FirstSeen) {
					existing.FirstSeen = subject.FirstSeen
				}

				if lastSeen.After(existing.LastSeen) {
					existing.LastSeen = lastSeen
				}
			} else {
				nodeMap[nextLevel] = &SubjectNode{
					Name:         nextLevel,
					IsLeaf:       isLeaf,
					MessageCount: subject.MessageCount.Load(),
					FirstSeen:    subject.FirstSeen,
					LastSeen:     lastSeen,
				}
			}
		}
	}

	// Convert map to slice
	var nodes []SubjectNode
	for _, node := range nodeMap {
		nodes = append(nodes, *node)
	}

	// Sort alphabetically to maintain consistent order
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return nodes
}
