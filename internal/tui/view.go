// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// getPauseResumeLabel returns the key and label for pause/resume control
func (m Model) getPauseResumeLabel() (key string, label string) {
	key = "<s>"
	if m.viewer != nil && m.viewer.IsPaused() {
		label = "resume"
	} else {
		label = "pause"
	}
	return key, label
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Wait for initial window size before rendering
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Render header and command bar first to measure their heights
	header := m.renderHeader()
	commandBar := m.renderCommandBar()

	// Calculate available height for content based on actual component heights
	headerHeight := lipgloss.Height(header)
	commandBarHeight := lipgloss.Height(commandBar)
	contentHeight := m.height - headerHeight - commandBarHeight - 1 // Reduce by 1 to prevent header overflow

	// When command bar is active, header is 1 line shorter, so reduce content by 1 to maintain same size
	if m.commandBarActive {
		contentHeight--
	}

	// Ensure we don't create content that's too tall
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Build content based on current view mode
	var content string
	switch m.viewMode {
	case ViewModeMessageInspect:
		content = m.renderMessageInspectWithHeight(contentHeight)
	case ViewModeLog:
		content = m.renderLogViewWithHeight(contentHeight)
	default:
		content = m.renderContentWithHeight(contentHeight)
	}

	// Combine all sections
	if m.commandBarActive {
		return lipgloss.JoinVertical(lipgloss.Left, header, commandBar, content)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

// renderHeader creates the header bar with app info and status
func (m Model) renderHeader() string {
	// Handle very small widths with simplified header
	if IsNarrowTerminal(m.width) {
		status := "●"
		if m.IsConnected() {
			status = HeaderConnectedStyle.Render(status)
		} else {
			status = HeaderDisconnectedStyle.Render(status)
		}
		simpleHeader := fmt.Sprintf("NLS %s | q:quit", status)
		return HeaderContainerStyle.
			Width(m.width).
			Padding(0, 1).
			Render(simpleHeader)
	}

	// ASCII art logo
	logo := HeaderAppNameStyle.Render(Logo)

	// Connection status
	var statusText string
	var statusStyle lipgloss.Style
	if m.IsConnected() {
		statusStyle = HeaderConnectedStyle
		statusText = "● Connected"
	} else {
		statusStyle = HeaderDisconnectedStyle
		statusText = "● Disconnected"
	}

	status := statusStyle.Render(statusText)
	server := HeaderServerStyle.Render(fmt.Sprintf("Server: %s", m.serverURL))

	// Add view mode indicator for browse mode
	var viewModeText string
	if m.viewMode == ViewModeBrowse {
		if m.flatViewMode {
			viewModeText = HeaderServerStyle.Render("View: flat")
		} else {
			viewModeText = HeaderServerStyle.Render("View: hierarchical")
		}
	}

	statusLines := []string{"", status, server}
	if viewModeText != "" {
		statusLines = append(statusLines, viewModeText)
	}
	// Add spacing at bottom - reduce by one line when command bar is active
	if m.commandBarActive {
		statusLines = append(statusLines, "")
	} else {
		statusLines = append(statusLines, "", "")
	}

	statusInfo := HeaderStatusInfoStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		statusLines...,
	))

	var controls1, controlsInfo1, controls2, controlsInfo2 string

	switch m.viewMode {
	case ViewModeMessageInspect:
		// View message inspect controls
		pauseKey, pauseLabel := m.getPauseResumeLabel()

		controls1 = HeaderControlStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"<esc>",
			"<↑↓>",
			"<←→>",
		))

		controlsInfo1 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"back",
			"scroll",
			"prev/next",
		))

		controls2 = HeaderControlStyle.
			MarginLeft(3).
			Render(lipgloss.JoinVertical(
				lipgloss.Left,
				"",
				"<g/G>",
				pauseKey,
				"<q>",
			))

		controlsInfo2 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"top/bottom",
			pauseLabel,
			"quit",
		))
	case ViewModeLog:
		// Log view controls
		pauseKey, pauseLabel := m.getPauseResumeLabel()

		controls1 = HeaderControlStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"<esc>",
			"<↑↓>",
			"<enter>",
		))

		controlsInfo1 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"back",
			"scroll",
			"inspect",
		))

		controls2 = HeaderControlStyle.
			MarginLeft(3).
			Render(lipgloss.JoinVertical(
				lipgloss.Left,
				"",
				"<g/G>",
				pauseKey,
				"<q>",
			))

		controlsInfo2 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"top/bottom",
			pauseLabel,
			"quit",
		))
	default:
		// Browse view controls
		controls1 = HeaderControlStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"<enter>",
			"<esc>",
			"<c>",
		))

		controlsInfo1 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"select",
			"back",
			"toggle view",
		))

		controls2 = HeaderControlStyle.
			MarginLeft(3).
			Render(lipgloss.JoinVertical(
				lipgloss.Left,
				"",
				"<l>",
				"<:>",
				"<q>",
			))

		controlsInfo2 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"logs",
			"filter",
			"quit",
		))
	}

	// Combine logo and status horizontally
	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logo,
		statusInfo,
		controls1,
		controlsInfo1,
		controls2,
		controlsInfo2,
	)

	// Apply container style with padding and width
	// Width sets content area, so account for horizontal padding (1 left + 1 right = 2)
	return HeaderContainerStyle.
		Width(m.width-2).
		Padding(0, 1).
		Render(headerContent)
}

// renderCommandBar creates the command input bar
func (m Model) renderCommandBar() string {
	if !m.commandBarActive {
		return ""
	}

	// Show filter prompt with instructions
	var promptText string
	if m.commandInput == "" {
		promptText = "Filter: (type to filter subjects, Enter to apply, Esc to cancel)"
	} else {
		promptText = fmt.Sprintf("Filter: %s", m.commandInput)
	}

	prompt := CommandBarStyle.
		Width(m.width).
		Render(promptText)
	return prompt
}
