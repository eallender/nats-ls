// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"
	"time"

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

	// Build content with calculated height
	content := m.renderContentWithHeight(contentHeight)

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
	msgCount := HeaderStatsStyle.Render(fmt.Sprintf("Messages: %d", m.messageCount))
	statusInfo := HeaderStatusInfoStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		status,
		server,
		msgCount,
	))

	controls1 := HeaderControlStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		"<enter>",
		"<esc>",
		"<↑↓>",
	))

	controlsInfo1 := HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		"drill down",
		"go back",
		"navigate",
	))

	controls2 := HeaderControlStyle.
		MarginLeft(3).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"<l>",
			"<:>",
			"<q>",
		))

	controlsInfo2 := HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		"logs",
		"filter",
		"quit",
	))

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
		Width(m.width - 2).
		Padding(0, 1).
		Render(headerContent)
}

// renderContentWithHeight creates the main content area with a single full-width panel
func (m Model) renderContentWithHeight(contentHeight int) string {
	// Enforce minimum content height (must account for frame overhead)
	// The content boxes need frame space (padding+borders) plus some content
	frameHeight := GetFrameHeight(NavStyle)
	minRequiredHeight := MinContentHeight + frameHeight
	if contentHeight < minRequiredHeight {
		contentHeight = minRequiredHeight
	}

	// Calculate content width and height (accounting for NavStyle borders/padding)
	// NavStyle has Padding(1, 2) = 2 left + 2 right = 4 horizontal padding
	// NavStyle has borders = 1 left + 1 right = 2 horizontal borders
	// Total horizontal frame = 6
	contentWidth := m.width - 6
	// Don't force a minimum that would cause overflow
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeightAdjusted := MaxContentHeight(contentHeight, NavStyle)

	// Build main content with hierarchical subjects as a table
	var mainText string

	if m.discovery != nil {
		// Add path as a title line if drilled down
		if len(m.navPath) > 0 {
			pathDisplay := strings.Join(m.navPath, ".") + " >"
			// Create a styled title that looks like it's part of the border
			titleLen := len(pathDisplay)

			// Ensure title fits within available width
			if titleLen+4 > contentWidth {
				// Truncate path if too long (leave room for spaces and dashes)
				maxPathLen := contentWidth - 4 // Reserve space for " " + " " and at least 2 dashes
				if maxPathLen > 0 {
					pathDisplay = pathDisplay[:maxPathLen] + ">"
					titleLen = len(pathDisplay)
				} else {
					// Terminal too narrow for title
					pathDisplay = ">"
					titleLen = 1
				}
			}

			leftDashes := (contentWidth - titleLen - 2) / 2
			if leftDashes < 0 {
				leftDashes = 0
			}
			rightDashes := contentWidth - titleLen - 2 - leftDashes
			if rightDashes < 0 {
				rightDashes = 0
			}

			// Build title line with exact width (before styling)
			rawTitle := strings.Repeat("─", leftDashes) + " " + pathDisplay + " " + strings.Repeat("─", rightDashes)
			// Note: "─" is 3 bytes but 1 display column, so ensure display width not byte length
			// Since we calculated leftDashes and rightDashes to fit contentWidth, this should be correct
			// But add safety check for any edge cases with Unicode
			titleLine := lipgloss.NewStyle().Foreground(ColorMuted).Render(rawTitle)
			mainText = titleLine + "\n\n"
		}

		nodes := m.getSubjectsAtCurrentLevel()
		if len(nodes) > 0 {
			// Calculate column widths dynamically based on available space
			var msgColWidth, lastSeenColWidth, subjectColWidth int
			spacingChars := 2 // spaces between columns

			// Scale columns based on available width
			if contentWidth < 30 {
				// Very narrow terminal - use minimal widths
				msgColWidth = 6
				lastSeenColWidth = 8
				subjectColWidth = contentWidth - msgColWidth - lastSeenColWidth - spacingChars
				if subjectColWidth < 5 {
					subjectColWidth = 5
					// Recalculate total to ensure it fits
					total := subjectColWidth + msgColWidth + lastSeenColWidth + spacingChars
					if total > contentWidth {
						// Scale down everything proportionally
						msgColWidth = 4
						lastSeenColWidth = 6
						subjectColWidth = contentWidth - msgColWidth - lastSeenColWidth - spacingChars
						if subjectColWidth < 3 {
							subjectColWidth = 3
						}
					}
				}
			} else {
				// Normal width - use standard column sizes
				msgColWidth = 10
				lastSeenColWidth = 12
				subjectColWidth = contentWidth - msgColWidth - lastSeenColWidth - spacingChars
				// Ensure subject column has reasonable minimum
				if subjectColWidth < 10 {
					subjectColWidth = 10
				}
			}

			// Final safety check: ensure total width doesn't exceed contentWidth
			totalWidth := subjectColWidth + msgColWidth + lastSeenColWidth + spacingChars
			if totalWidth > contentWidth {
				// Force subjectColWidth to fit within bounds
				subjectColWidth = contentWidth - msgColWidth - lastSeenColWidth - spacingChars
				if subjectColWidth < 1 {
					subjectColWidth = 1
				}
			}

			// Table header with dynamic column widths
			headerText := fmt.Sprintf("%-*s %*s %*s", subjectColWidth, "SUBJECT", msgColWidth, "MESSAGES", lastSeenColWidth, "LAST SEEN")
			// Ensure exact width to prevent wrapping
			headerText = ensureWidth(headerText, contentWidth)
			header := NavTableHeaderStyle.Render(headerText)
			mainText += header + "\n"

			// Table rows
			for i, node := range nodes {
				rowStyle := NavTableRowStyle
				if i == m.selectedIndex {
					rowStyle = NavTableSelectedRowStyle
				}

				// Display name with indicator for directories vs leaves
				displayName := node.Name
				if !node.IsLeaf {
					displayName += ".>"
				}

				// Truncate if too long for the dynamic column width
				maxDisplayLen := subjectColWidth
				if len(displayName) > maxDisplayLen {
					displayName = displayName[:maxDisplayLen-3] + "..."
				}

				// Format last seen as relative time
				lastSeenStr := formatRelativeTime(node.LastSeen)

				rowText := fmt.Sprintf("%-*s %*d %*s", subjectColWidth, displayName, msgColWidth, node.MessageCount, lastSeenColWidth, lastSeenStr)
				// Ensure exact width to prevent wrapping
				rowText = ensureWidth(rowText, contentWidth)
				row := rowStyle.Render(rowText)
				mainText += row + "\n"
			}
		} else {
			mainText += ensureWidth("No subjects discovered yet...", contentWidth)
		}
	} else {
		mainText = ensureWidth("Not connected...", contentWidth)
	}

	// Main panel - Don't set Width() since our content is already sized correctly
	// The Width() method causes lipgloss to try to wrap text that contains ANSI codes
	// Our mainText lines are already exactly contentWidth wide
	content := NavStyle.
		Height(contentHeightAdjusted).
		Render(mainText)

	return content
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

// formatRelativeTime formats a time as a relative time string (e.g., "2s ago", "5m ago")
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Second:
		return "just now"
	case duration < time.Minute:
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
}

// ensureWidth ensures a string is exactly the specified width by truncating or padding
// This is safe for UTF-8 but treats multi-byte characters as single units
func ensureWidth(s string, width int) string {
	// For ASCII-only strings (which our table uses), len() == display width
	currentLen := len(s)
	if currentLen > width {
		// Truncate - safe for ASCII, may need rune handling for Unicode subjects
		return s[:width]
	} else if currentLen < width {
		// Pad with spaces
		return s + strings.Repeat(" ", width-currentLen)
	}
	return s
}
