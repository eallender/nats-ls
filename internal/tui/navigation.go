// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"sort"
	"strings"
)

// SubjectNode represents a subject or subject prefix in the hierarchy
type SubjectNode struct {
	Name         string
	IsLeaf       bool // true if this is a complete subject, false if it's a prefix
	MessageCount int64
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

		// Get the part after the current prefix
		remainder := strings.TrimPrefix(subject.Name, currentPrefix)

		// Split by "." to get the next level
		parts := strings.Split(remainder, ".")

		if len(parts) > 0 && parts[0] != "" {
			nextLevel := parts[0]
			isLeaf := len(parts) == 1

			if existing, ok := nodeMap[nextLevel]; ok {
				// Aggregate message counts
				existing.MessageCount += subject.MessageCount.Load()
				// If any subject is a leaf, mark it as such
				if isLeaf {
					existing.IsLeaf = true
				}
			} else {
				nodeMap[nextLevel] = &SubjectNode{
					Name:         nextLevel,
					IsLeaf:       isLeaf,
					MessageCount: subject.MessageCount.Load(),
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
