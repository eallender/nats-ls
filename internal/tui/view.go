// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

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
	contentHeight := m.height - headerHeight - commandBarHeight

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
	layout := NewLayout(m.width, m.height)
	if layout.IsNarrow() {
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
	statusInfo := HeaderStatusInfoStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		status,
		server,
		"",
	))

	var controls1, controlsInfo1, controls2, controlsInfo2 string

	if m.viewMode == ViewModeMessageInspect {
		// Message inspect view controls
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
				"<q>",
				"",
			))

		controlsInfo2 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"top/bottom",
			"quit",
			"",
		))
	} else if m.viewMode == ViewModeLog {
		// Log view controls
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
				"<q>",
				"",
			))

		controlsInfo2 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"top/bottom",
			"quit",
			"",
		))
	} else {
		// Browse view controls
		controls1 = HeaderControlStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"<enter>",
			"<esc>",
			"<↑↓>",
		))

		controlsInfo1 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"select",
			"back",
			"navigate",
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

	prompt := CommandBarStyle.
		Width(m.width).
		Render(fmt.Sprintf(":%s", m.commandInput))
	return prompt
}
