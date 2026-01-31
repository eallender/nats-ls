// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If command bar is active, handle its input
		if m.commandBarActive {
			switch msg.String() {
			case "enter":
				// TODO: Implement command processing (e.g., filter subjects by pattern)
				// For now, just clear the command bar
				m.commandBarActive = false
				m.commandInput = ""
			case "esc":
				m.commandBarActive = false
				m.commandInput = ""
			case "backspace":
				if len(m.commandInput) > 0 {
					m.commandInput = m.commandInput[:len(m.commandInput)-1]
				}
			default:
				// Add character to input
				m.commandInput += msg.String()
			}
			return m, nil
		}

		// Log view mode key handling
		if m.viewMode == ViewModeLog {
			return m.handleLogViewKeys(msg)
		}

		// Browse mode key handling
		switch msg.String() {
		case ":":
			m.commandBarActive = true
			m.commandInput = ""
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "down", "j":
			nodes := m.getSubjectsAtCurrentLevel()
			if m.selectedIndex < len(nodes)-1 {
				m.selectedIndex++
			}
		case "enter":
			// Drill down into the selected subject
			nodes := m.getSubjectsAtCurrentLevel()
			if len(nodes) > 0 && m.selectedIndex < len(nodes) {
				selectedNode := nodes[m.selectedIndex]
				// Only drill down if it's not a leaf (i.e., has children)
				if !selectedNode.IsLeaf {
					m.navPath = append(m.navPath, selectedNode.Name)
					m.selectedIndex = 0
				}
			}
		case "l":
			// Switch to log view for the selected subject
			nodes := m.getSubjectsAtCurrentLevel()
			if len(nodes) > 0 && m.selectedIndex < len(nodes) && m.viewer != nil {
				selectedNode := nodes[m.selectedIndex]
				// Build the full subject path
				var subjectPath string
				if len(m.navPath) > 0 {
					subjectPath = strings.Join(m.navPath, ".") + "." + selectedNode.Name
				} else {
					subjectPath = selectedNode.Name
				}
				// Add wildcard if not a leaf to capture all messages under this prefix
				if !selectedNode.IsLeaf {
					subjectPath += ".>"
				}
				// Start watching the subject
				m.viewer.Watch(subjectPath)
				m.watchingSubject = subjectPath
				m.viewMode = ViewModeLog
				m.logScrollOffset = 0
			}
		case "esc":
			// Go back up one level
			if len(m.navPath) > 0 {
				m.navPath = m.navPath[:len(m.navPath)-1]
				m.selectedIndex = 0
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case connectAttemptMsg:
		if msg.err != nil {
			// Connection failed, retry after a delay
			return m, tickCmd
		}
		// Connection successful, update model
		m.nc = msg.nc
		m.viewer = msg.viewer
		m.discovery = msg.discovery
		// Start the tick loop to refresh the UI
		return m, tickCmd
	case tickMsg:
		// If not connected, try to reconnect
		if !m.IsConnected() {
			return m, tea.Batch(m.tryConnect, tickCmd)
		}
		// Otherwise just refresh the UI periodically to show new subjects
		return m, tickCmd
	}
	return m, nil
}

// handleLogViewKeys handles key presses when in log view mode
func (m Model) handleLogViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		// Go back to browse view
		if m.viewer != nil {
			m.viewer.Watch("") // Stop watching
		}
		m.viewMode = ViewModeBrowse
		m.watchingSubject = ""
		m.logScrollOffset = 0
	case "up", "k":
		// Scroll up in log view (show older messages)
		if m.viewer != nil {
			msgCount := m.viewer.GetMessageCount()
			if m.logScrollOffset < msgCount-1 {
				m.logScrollOffset++
			}
		}
	case "down", "j":
		// Scroll down in log view (show newer messages)
		if m.logScrollOffset > 0 {
			m.logScrollOffset--
		}
	case "g":
		// Jump to top (oldest messages)
		if m.viewer != nil {
			msgCount := m.viewer.GetMessageCount()
			if msgCount > 0 {
				m.logScrollOffset = msgCount - 1
			}
		}
	case "G":
		// Jump to bottom (most recent messages)
		m.logScrollOffset = 0
	}
	return m, nil
}
