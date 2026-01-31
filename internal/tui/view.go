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

	// Build content based on current view mode
	var content string
	switch m.viewMode {
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
	msgCount := HeaderStatsStyle.Render(fmt.Sprintf("Messages: %d", m.messageCount))
	statusInfo := HeaderStatusInfoStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		status,
		server,
		msgCount,
	))

	var controls1, controlsInfo1, controls2, controlsInfo2 string

	if m.viewMode == ViewModeLog {
		// Log view controls
		controls1 = HeaderControlStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"<esc>",
			"<↑↓>",
			"<g/G>",
		))

		controlsInfo1 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"back",
			"scroll",
			"top/bottom",
		))

		controls2 = HeaderControlStyle.
			MarginLeft(3).
			Render(lipgloss.JoinVertical(
				lipgloss.Left,
				"",
				"<q>",
				"",
				"",
			))

		controlsInfo2 = HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			"quit",
			"",
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
			"drill down",
			"go back",
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

	// Determine border title if drilled down
	var borderTitle string
	if m.discovery != nil && len(m.navPath) > 0 {
		borderTitle = strings.Join(m.navPath, ".") + ".>"
		// Truncate if too long
		maxTitleLen := contentWidth - 4
		if len(borderTitle) > maxTitleLen && maxTitleLen > 3 {
			borderTitle = borderTitle[:maxTitleLen-3] + "...>"
		}
	}

	if m.discovery != nil {
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
	boxStyle := NavStyle.Height(contentHeightAdjusted)

	// Add border title if we have a path
	if borderTitle != "" {
		styledTitle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(borderTitle)
		boxStyle = boxStyle.BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
		content := boxStyle.Render(mainText)
		// Insert title into top border
		// Border width = content + horizontal padding (4) + borders (2) = contentWidth + 6
		content = insertBorderTitle(content, styledTitle, contentWidth+6)
		return content
	}

	return boxStyle.Render(mainText)
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

// insertBorderTitle inserts a centered title into the top border of a rendered box
// borderWidth should be the expected total width of the border (including corners)
func insertBorderTitle(rendered, title string, borderWidth int) string {
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	if borderWidth < 4 {
		return rendered
	}

	// Calculate positioning for centered title
	titleDisplayWidth := lipgloss.Width(title)
	availableWidth := borderWidth - 2                      // minus corners
	dashesNeeded := availableWidth - titleDisplayWidth - 2 // -2 for spaces around title

	if dashesNeeded < 2 {
		return rendered
	}

	// Center the title
	leftDashes := dashesNeeded / 2
	rightDashes := dashesNeeded - leftDashes

	// Build the new top border with proper box-drawing characters and consistent styling
	borderStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	newTopBorder := borderStyle.Render("┌"+strings.Repeat("─", leftDashes)) +
		" " + title + " " +
		borderStyle.Render(strings.Repeat("─", rightDashes)+"┐")

	lines[0] = newTopBorder

	return strings.Join(lines, "\n")
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

// renderLogViewWithHeight creates the log view showing live NATS messages
func (m Model) renderLogViewWithHeight(contentHeight int) string {
	// Enforce minimum content height
	frameHeight := GetFrameHeight(LogViewStyle)
	minRequiredHeight := MinContentHeight + frameHeight
	if contentHeight < minRequiredHeight {
		contentHeight = minRequiredHeight
	}

	// Calculate content dimensions
	contentWidth := m.width - 6
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeightAdjusted := MaxContentHeight(contentHeight, LogViewStyle)

	// Border title for the log view
	borderTitle := fmt.Sprintf("Watching: %s", m.watchingSubject)
	maxTitleLen := contentWidth - 4
	if len(borderTitle) > maxTitleLen && maxTitleLen > 3 {
		borderTitle = borderTitle[:maxTitleLen-3] + "..."
	}

	var logText string

	if m.viewer == nil {
		logText = LogEmptyStyle.Render("Viewer not available")
	} else {
		messages := m.viewer.GetMessages()
		if len(messages) == 0 {
			logText += LogEmptyStyle.Render("Waiting for messages...")
		} else {
			// Calculate how many lines we can show
			// Lines used: scroll indicator(1) + 1 extra to prevent overflow = 2
			availableLines := contentHeightAdjusted - 2
			if availableLines < 1 {
				availableLines = 1
			}

			// Messages flow from bottom to top (newest at bottom)
			// scrollOffset=0 means showing the most recent messages at the bottom
			// scrollOffset>0 means scrolling back in history (older messages)
			endIdx := len(messages) - m.logScrollOffset
			if endIdx < 0 {
				endIdx = 0
			}
			if endIdx > len(messages) {
				endIdx = len(messages)
			}
			startIdx := endIdx - availableLines
			if startIdx < 0 {
				startIdx = 0
			}

			// Calculate actual number of messages to display
			numMessages := endIdx - startIdx

			// Add blank lines if we have fewer messages than available space
			// This pushes messages to the bottom of the view
			if numMessages < availableLines {
				blankLines := availableLines - numMessages
				for i := 0; i < blankLines; i++ {
					logText += "\n"
				}
			}

			for i := startIdx; i < endIdx; i++ {
				msg := messages[i]

				// Format timestamp
				timestamp := msg.Timestamp.Format("15:04:05.000")

				// Format subject (may differ from watched subject if using wildcard)
				subject := msg.Subject

				// Format data - try to display as string, truncate if needed
				data := string(msg.Data)
				// Replace newlines with spaces for single-line display
				data = strings.ReplaceAll(data, "\n", " ")
				data = strings.ReplaceAll(data, "\r", "")

				// Calculate available width for data
				// Format: "HH:MM:SS.mmm | subject | data"
				timestampWidth := 12
				separatorWidth := 6 // " | " twice
				subjectWidth := len(subject)
				if subjectWidth > 30 {
					subjectWidth = 30
					subject = subject[:27] + "..."
				}
				dataWidth := contentWidth - timestampWidth - separatorWidth - subjectWidth
				if dataWidth < 10 {
					dataWidth = 10
				}

				// Truncate data if needed
				if len(data) > dataWidth {
					data = data[:dataWidth-3] + "..."
				}

				// Build the log line (no trailing newline on last message)
				line := fmt.Sprintf("%s │ %s │ %s",
					LogTimestampStyle.Render(timestamp),
					LogSubjectStyle.Render(subject),
					LogDataStyle.Render(data),
				)
				if i < endIdx-1 {
					logText += line + "\n"
				} else {
					logText += line
				}
			}

			// Show scroll indicator on its own line
			scrollInfo := fmt.Sprintf("── %d/%d messages ──", endIdx, len(messages))
			logText += "\n" + LogTimestampStyle.Render(scrollInfo)
		}
	}

	// Pad logText to ensure consistent width (like nav view does with ensureWidth)
	// This prevents the box from resizing based on content
	logLines := strings.Split(logText, "\n")
	for i, line := range logLines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			logLines[i] = line + strings.Repeat(" ", contentWidth-lineWidth)
		}
	}
	logText = strings.Join(logLines, "\n")

	// Render in the log view style with border title
	boxStyle := LogViewStyle.Height(contentHeightAdjusted)
	boxStyle = boxStyle.BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
	content := boxStyle.Render(logText)

	// Insert the border title with same styling as nav view
	// Border width = content width + padding (2+2) + borders (1+1) = contentWidth + 6
	// But NavStyle also uses contentWidth+2, let's match that approach
	styledTitle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(borderTitle)
	// The actual border width = content + horizontal padding (4) + borders (2) = contentWidth + 6
	content = insertBorderTitle(content, styledTitle, contentWidth+6)

	return content
}
