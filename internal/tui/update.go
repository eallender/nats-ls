// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// getBrowseAvailableLines calculates how many subject rows can be displayed
func (m Model) getBrowseAvailableLines() int {
	// Calculate exactly how view_browse.go does it
	// This must match the rendering logic to prevent selection going off-screen
	header := m.renderHeader()
	commandBar := m.renderCommandBar()
	headerHeight := lipgloss.Height(header)
	commandBarHeight := lipgloss.Height(commandBar)
	contentHeight := m.height - headerHeight - commandBarHeight - 1

	if m.commandBarActive {
		contentHeight--
	}

	if contentHeight < 1 {
		contentHeight = 1
	}

	// This matches view_browse.go's calculation
	innerContentHeight := MaxContentHeight(contentHeight, NavStyle)
	availableLines := innerContentHeight - 2 // header + scroll indicator/stats line
	if availableLines < 1 {
		availableLines = 1
	}
	return availableLines
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If command bar is active, handle its input
		if m.commandBarActive {
			switch msg.String() {
			case "enter":
				// Accept the filter and close command bar
				m.commandBarActive = false
				m.commandInput = ""
				// Reset scroll position when filter is applied
				m.selectedIndex = 0
				m.browseScrollOffset = 0
			case "esc":
				// Restore previous filter and close command bar
				m.subjectFilter = m.commandPreviousFilter
				m.commandBarActive = false
				m.commandInput = ""
				m.selectedIndex = 0
				m.browseScrollOffset = 0
			case "backspace":
				if len(m.commandInput) > 0 {
					m.commandInput = m.commandInput[:len(m.commandInput)-1]
					// Apply filter live
					(&m).applyLiveFilter()
				}
			default:
				// Add character to input and apply filter live
				m.commandInput += msg.String()
				(&m).applyLiveFilter()
			}
			return m, nil
		}

		// Message inspect mode key handling
		if m.viewMode == ViewModeMessageInspect {
			return m.handleMessageInspectKeys(msg)
		}

		// Log view mode key handling
		if m.viewMode == ViewModeLog {
			return m.handleLogViewKeys(msg)
		}

		// Browse mode key handling
		switch msg.String() {
		case ":":
			m.commandBarActive = true
			// Save current filter state and pre-populate input
			m.commandPreviousFilter = m.subjectFilter
			m.commandInput = m.subjectFilter
		case "c":
			// Toggle between flat and hierarchical view
			m.flatViewMode = !m.flatViewMode
			// Clear navigation path when entering flat view
			if m.flatViewMode {
				m.navPath = nil
			}
			// Reset selection and scroll
			m.selectedIndex = 0
			m.browseScrollOffset = 0
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				// Scroll up if needed to keep selection visible
				if m.selectedIndex < m.browseScrollOffset {
					m.browseScrollOffset = m.selectedIndex
				}
			}
		case "down", "j":
			nodes := m.getSubjectsAtCurrentLevel()
			if m.selectedIndex < len(nodes)-1 {
				m.selectedIndex++
				// Scroll down if needed to keep selection visible
				availableLines := m.getBrowseAvailableLines()
				if m.selectedIndex >= m.browseScrollOffset+availableLines {
					m.browseScrollOffset = m.selectedIndex - availableLines + 1
				}
			}
		case "enter":
			// Drill down into the selected subject or view logs for leaf nodes
			nodes := m.getSubjectsAtCurrentLevel()
			if len(nodes) > 0 && m.selectedIndex < len(nodes) {
				selectedNode := nodes[m.selectedIndex]
				if !selectedNode.IsLeaf {
					// Drill down if it's not a leaf (i.e., has children)
					m.navPath = append(m.navPath, selectedNode.Name)
					m.selectedIndex = 0
					m.browseScrollOffset = 0
				} else if m.viewer != nil {
					// Switch to log view for leaf nodes
					var subjectPath string
					if len(m.navPath) > 0 {
						subjectPath = strings.Join(m.navPath, ".") + "." + selectedNode.Name
					} else {
						subjectPath = selectedNode.Name
					}
					// Start watching the subject
					m.viewer.Watch(subjectPath)
					m.watchingSubject = subjectPath
					m.viewMode = ViewModeLog
					m.logScrollOffset = 0
					m.logSelectedIndex = 0
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
				m.logSelectedIndex = 0
			}
		case "esc":
			// Go back up one level
			if len(m.navPath) > 0 {
				m.navPath = m.navPath[:len(m.navPath)-1]
				m.selectedIndex = 0
				m.browseScrollOffset = 0
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

// applyLiveFilter applies the current command input as a filter in real-time
func (m *Model) applyLiveFilter() {
	input := strings.TrimSpace(m.commandInput)

	// Direct input becomes the filter (no need for "filter" prefix while typing)
	m.subjectFilter = input

	// Reset selection when filter changes
	m.selectedIndex = 0
	m.browseScrollOffset = 0
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
		m.logSelectedIndex = 0
	case "enter":
		// Enter message inspect mode for the selected message
		if m.viewer != nil {
			messages := m.viewer.GetMessages()
			if len(messages) > 0 {
				// Calculate the actual message index from the selection
				// logSelectedIndex is relative to visible messages (0 = newest visible)
				m.inspectedMessageIndex = len(messages) - 1 - m.logScrollOffset - m.logSelectedIndex
				if m.inspectedMessageIndex < 0 {
					m.inspectedMessageIndex = 0
				}
				if m.inspectedMessageIndex >= len(messages) {
					m.inspectedMessageIndex = len(messages) - 1
				}
				m.inspectScrollOffset = 0
				m.viewMode = ViewModeMessageInspect
			}
		}
	case "up", "k":
		// Move selection up (to older messages)
		if m.viewer != nil {
			messages := m.viewer.GetMessages()
			msgCount := len(messages)
			if msgCount > 0 {
				// Calculate visible lines (approximation - matches view_log.go logic)
				availableLines := m.getLogViewAvailableLines()

				// Calculate how many messages are above the current view
				endIdx := msgCount - m.logScrollOffset
				if endIdx > msgCount {
					endIdx = msgCount
				}
				startIdx := endIdx - availableLines
				if startIdx < 0 {
					startIdx = 0
				}
				visibleCount := endIdx - startIdx

				// Move selection up within visible range
				if m.logSelectedIndex < visibleCount-1 {
					m.logSelectedIndex++
				} else if startIdx > 0 {
					// Scroll up to show more messages
					m.logScrollOffset++
				}
			}
		}
	case "down", "j":
		// Move selection down (to newer messages)
		if m.logSelectedIndex > 0 {
			m.logSelectedIndex--
		} else if m.logScrollOffset > 0 {
			// Scroll down to show newer messages
			m.logScrollOffset--
		}
	case "g":
		// Jump to top (oldest messages)
		if m.viewer != nil {
			msgCount := m.viewer.GetMessageCount()
			if msgCount > 0 {
				availableLines := m.getLogViewAvailableLines()
				m.logScrollOffset = msgCount - availableLines
				if m.logScrollOffset < 0 {
					m.logScrollOffset = 0
				}
				// Select the topmost (oldest) visible message
				visibleCount := availableLines
				if visibleCount > msgCount {
					visibleCount = msgCount
				}
				m.logSelectedIndex = visibleCount - 1
			}
		}
	case "G":
		// Jump to bottom (most recent messages)
		m.logScrollOffset = 0
		m.logSelectedIndex = 0
	case "s":
		// Toggle pause/resume autoscroll
		if m.viewer != nil {
			if m.viewer.IsPaused() {
				m.viewer.Resume()
				// Reset to latest messages when resuming
				m.logScrollOffset = 0
				m.logSelectedIndex = 0
			} else {
				m.viewer.Pause()
			}
		}
	}
	return m, nil
}

// getLogViewAvailableLines calculates how many log lines can be displayed
func (m Model) getLogViewAvailableLines() int {
	// This should match the logic in view_log.go
	frameHeight := GetFrameHeight(LogViewStyle)
	headerHeight := 5 // Approximate header height
	contentHeight := m.height - headerHeight
	minRequiredHeight := MinContentHeight + frameHeight
	if contentHeight < minRequiredHeight {
		contentHeight = minRequiredHeight
	}
	contentHeightAdjusted := MaxContentHeight(contentHeight, LogViewStyle)
	availableLines := contentHeightAdjusted - 2 // scroll indicator + buffer
	if availableLines < 1 {
		availableLines = 1
	}
	return availableLines
}

// handleMessageInspectKeys handles key presses when in message inspect mode
func (m Model) handleMessageInspectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "esc":
		// Go back to log view
		m.viewMode = ViewModeLog
		m.inspectScrollOffset = 0
	case "up", "k":
		// Scroll up in the message content
		if m.inspectScrollOffset > 0 {
			m.inspectScrollOffset--
		}
	case "down", "j":
		// Scroll down in the message content
		m.inspectScrollOffset++
	case "g":
		// Jump to top
		m.inspectScrollOffset = 0
	case "G":
		// Jump to bottom - will be clamped in render
		m.inspectScrollOffset = 99999
	case "left", "h":
		// Navigate to previous message
		if m.inspectedMessageIndex > 0 {
			m.inspectedMessageIndex--
			m.inspectScrollOffset = 0
		}
	case "right", "l":
		// Navigate to next message
		if m.viewer != nil {
			messages := m.viewer.GetMessages()
			if m.inspectedMessageIndex < len(messages)-1 {
				m.inspectedMessageIndex++
				m.inspectScrollOffset = 0
			}
		}
	case "s":
		// Toggle pause/resume
		if m.viewer != nil {
			if m.viewer.IsPaused() {
				m.viewer.Resume()
			} else {
				m.viewer.Pause()
			}
		}
	}
	return m, nil
}
