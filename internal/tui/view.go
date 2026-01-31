// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"bytes"
	"encoding/json"
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

			// Show scroll indicator on its own line, centered
			var scrollInfo string
			if m.logScrollOffset == 0 {
				scrollInfo = "── latest ──"
			} else {
				scrollInfo = fmt.Sprintf("── %d newer ↓ ──", m.logScrollOffset)
			}
			// Center the scroll info
			scrollInfoWidth := len(scrollInfo)
			if scrollInfoWidth < contentWidth {
				padding := (contentWidth - scrollInfoWidth) / 2
				scrollInfo = strings.Repeat(" ", padding) + scrollInfo
			}
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

// renderMessageInspectWithHeight creates the message inspector view with scrollable pretty JSON
func (m Model) renderMessageInspectWithHeight(contentHeight int) string {
	// Enforce minimum content height
	frameHeight := GetFrameHeight(InspectViewStyle)
	minRequiredHeight := MinContentHeight + frameHeight
	if contentHeight < minRequiredHeight {
		contentHeight = minRequiredHeight
	}

	// Calculate content dimensions
	contentWidth := m.width - 6
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeightAdjusted := MaxContentHeight(contentHeight, InspectViewStyle)

	var inspectText string

	if m.viewer == nil {
		inspectText = LogEmptyStyle.Render("Viewer not available")
	} else {
		messages := m.viewer.GetMessages()
		if len(messages) == 0 || m.inspectedMessageIndex >= len(messages) {
			inspectText = LogEmptyStyle.Render("No message to inspect")
		} else {
			msg := messages[m.inspectedMessageIndex]

			// Build header with message metadata
			header := fmt.Sprintf("Subject: %s\nTime: %s\nSize: %d bytes\n",
				InspectKeyStyle.Render(msg.Subject),
				InspectHeaderStyle.Render(msg.Timestamp.Format("2006-01-02 15:04:05.000")),
				len(msg.Data),
			)

			// Add headers if present
			if len(msg.Headers) > 0 {
				header += "Headers:\n"
				for key, values := range msg.Headers {
					header += fmt.Sprintf("  %s: %s\n",
						InspectKeyStyle.Render(key),
						strings.Join(values, ", "),
					)
				}
			}

			header += strings.Repeat("─", contentWidth) + "\n"

			// Format the message data as pretty JSON if possible
			formattedData := formatMessageData(msg.Data, contentWidth)

			// Split into lines for scrolling
			allLines := strings.Split(header+formattedData, "\n")

			// Calculate available lines for content (minus navigation hint)
			availableLines := contentHeightAdjusted - 1
			if availableLines < 1 {
				availableLines = 1
			}

			// Clamp scroll offset
			maxScroll := len(allLines) - availableLines
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.inspectScrollOffset > maxScroll {
				m.inspectScrollOffset = maxScroll
			}

			// Get visible lines
			endIdx := m.inspectScrollOffset + availableLines
			if endIdx > len(allLines) {
				endIdx = len(allLines)
			}

			visibleLines := allLines[m.inspectScrollOffset:endIdx]

			// Pad lines to content width
			for i, line := range visibleLines {
				lineWidth := lipgloss.Width(line)
				if lineWidth < contentWidth {
					visibleLines[i] = line + strings.Repeat(" ", contentWidth-lineWidth)
				} else if lineWidth > contentWidth {
					// Truncate long lines
					visibleLines[i] = truncateToWidth(line, contentWidth)
				}
			}

			inspectText = strings.Join(visibleLines, "\n")

			// Add scroll indicator
			var scrollInfo string
			if maxScroll == 0 {
				scrollInfo = fmt.Sprintf("── message %d of %d ──", m.inspectedMessageIndex+1, len(messages))
			} else {
				scrollInfo = fmt.Sprintf("── message %d of %d │ line %d-%d of %d ──",
					m.inspectedMessageIndex+1, len(messages),
					m.inspectScrollOffset+1, endIdx, len(allLines))
			}
			scrollInfoWidth := len(scrollInfo)
			if scrollInfoWidth < contentWidth {
				padding := (contentWidth - scrollInfoWidth) / 2
				scrollInfo = strings.Repeat(" ", padding) + scrollInfo
			}
			inspectText += "\n" + InspectHeaderStyle.Render(scrollInfo)
		}
	}

	// Border title for the inspect view
	borderTitle := "Message Inspector"

	// Render in the inspect view style with border title
	boxStyle := InspectViewStyle.Height(contentHeightAdjusted)
	boxStyle = boxStyle.BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
	content := boxStyle.Render(inspectText)

	styledTitle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(borderTitle)
	content = insertBorderTitle(content, styledTitle, contentWidth+6)

	return content
}

// formatMessageData formats message data as pretty JSON with syntax highlighting
func formatMessageData(data []byte, maxWidth int) string {
	// Try to parse as JSON first
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		// Successfully parsed as JSON, format it nicely
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(jsonData); err == nil {
			formatted := strings.TrimSpace(buf.String())
			// Apply syntax highlighting, then wrap long lines
			highlighted := syntaxHighlightJSON(formatted)
			return wrapStyledLines(highlighted, maxWidth)
		}
	}

	// Not valid JSON - check if it's printable text or binary
	return formatRawData(data, maxWidth)
}

// wrapStyledLines wraps lines that exceed maxWidth, handling ANSI codes
func wrapStyledLines(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	lines := strings.Split(text, "\n")
	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		lineWidth := lipgloss.Width(line)
		if lineWidth <= maxWidth {
			result.WriteString(line)
			continue
		}

		// Line is too long, need to wrap it
		// For styled lines, we truncate and add continuation indicator
		result.WriteString(truncateToWidth(line, maxWidth))
	}

	return result.String()
}

// formatRawData formats non-JSON data, handling binary and text appropriately
func formatRawData(data []byte, maxWidth int) string {
	// Check if data is printable UTF-8 text
	if isPrintableText(data) {
		text := string(data)
		// Wrap long lines for readability
		return wrapText(text, maxWidth)
	}

	// Binary data - show hex dump
	return formatHexDump(data, maxWidth)
}

// isPrintableText checks if data is mostly printable UTF-8 text
func isPrintableText(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	printable := 0
	total := 0

	for i := 0; i < len(data); {
		r, size := rune(data[i]), 1
		if data[i] >= 0x80 {
			// Multi-byte UTF-8
			var ok bool
			r, size = decodeRune(data[i:])
			if !ok || r == '\uFFFD' {
				// Invalid UTF-8
				total++
				i++
				continue
			}
		} else {
			r = rune(data[i])
		}

		total++
		// Consider printable: letters, digits, punctuation, spaces, newlines, tabs
		if (r >= 32 && r < 127) || r == '\n' || r == '\r' || r == '\t' || (r >= 0x80 && r != '\uFFFD') {
			printable++
		}
		i += size
	}

	// If more than 90% is printable, treat as text
	return total > 0 && float64(printable)/float64(total) >= 0.9
}

// decodeRune decodes a UTF-8 rune from bytes
func decodeRune(data []byte) (rune, int) {
	if len(data) == 0 {
		return '\uFFFD', 0
	}

	// Simple UTF-8 decoding
	b := data[0]
	if b < 0x80 {
		return rune(b), 1
	}
	if b < 0xC0 {
		return '\uFFFD', 1
	}
	if b < 0xE0 && len(data) >= 2 {
		return rune(b&0x1F)<<6 | rune(data[1]&0x3F), 2
	}
	if b < 0xF0 && len(data) >= 3 {
		return rune(b&0x0F)<<12 | rune(data[1]&0x3F)<<6 | rune(data[2]&0x3F), 3
	}
	if len(data) >= 4 {
		return rune(b&0x07)<<18 | rune(data[1]&0x3F)<<12 | rune(data[2]&0x3F)<<6 | rune(data[3]&0x3F), 4
	}
	return '\uFFFD', 1
}

// wrapText wraps text at maxWidth characters
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		// Wrap long lines
		for len(line) > maxWidth {
			result.WriteString(line[:maxWidth])
			result.WriteString("\n")
			line = line[maxWidth:]
		}
		result.WriteString(line)
	}

	return result.String()
}

// formatHexDump formats binary data as a hex dump
func formatHexDump(data []byte, maxWidth int) string {
	var result strings.Builder
	bytesPerLine := 16

	// Adjust bytes per line based on available width
	// Format: "0000: XX XX XX XX  XX XX XX XX  |........|"
	// Minimum width needed: 6 (offset) + 3*8 + 2 (gap) + 3*8 + 2 (gap) + 10 (ascii) = ~70
	if maxWidth < 70 {
		bytesPerLine = 8
	}

	result.WriteString(InspectHeaderStyle.Render("Binary data - hex dump:") + "\n\n")

	for offset := 0; offset < len(data); offset += bytesPerLine {
		// Offset
		result.WriteString(InspectNumberStyle.Render(fmt.Sprintf("%04x", offset)))
		result.WriteString(InspectBracketStyle.Render(": "))

		// Hex bytes
		end := offset + bytesPerLine
		if end > len(data) {
			end = len(data)
		}

		for i := offset; i < offset+bytesPerLine; i++ {
			if i < end {
				result.WriteString(InspectKeyStyle.Render(fmt.Sprintf("%02x ", data[i])))
			} else {
				result.WriteString("   ")
			}
			if i == offset+bytesPerLine/2-1 {
				result.WriteString(" ")
			}
		}

		// ASCII representation
		result.WriteString(InspectBracketStyle.Render(" |"))
		for i := offset; i < end; i++ {
			if data[i] >= 32 && data[i] < 127 {
				result.WriteString(InspectStringStyle.Render(string(data[i])))
			} else {
				result.WriteString(InspectNullStyle.Render("."))
			}
		}
		result.WriteString(InspectBracketStyle.Render("|"))

		if offset+bytesPerLine < len(data) {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// syntaxHighlightJSON applies syntax highlighting to formatted JSON
func syntaxHighlightJSON(jsonStr string) string {
	var result strings.Builder
	inString := false
	inKey := false
	i := 0

	for i < len(jsonStr) {
		c := jsonStr[i]

		switch {
		case c == '"':
			if inString {
				// End of string
				if inKey {
					result.WriteString(InspectKeyStyle.Render("\""))
				} else {
					result.WriteString(InspectStringStyle.Render("\""))
				}
				inString = false
				inKey = false
			} else {
				// Start of string - check if it's a key
				inString = true
				// Look ahead to see if this is a key (followed by : after closing quote)
				inKey = isJSONKey(jsonStr, i)
				if inKey {
					result.WriteString(InspectKeyStyle.Render("\""))
				} else {
					result.WriteString(InspectStringStyle.Render("\""))
				}
			}
		case inString:
			// Inside a string, continue with current style
			if inKey {
				result.WriteString(InspectKeyStyle.Render(string(c)))
			} else {
				result.WriteString(InspectStringStyle.Render(string(c)))
			}
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ':' || c == ',':
			result.WriteString(InspectBracketStyle.Render(string(c)))
		case c == 't' && i+4 <= len(jsonStr) && jsonStr[i:i+4] == "true":
			result.WriteString(InspectBoolStyle.Render("true"))
			i += 3
		case c == 'f' && i+5 <= len(jsonStr) && jsonStr[i:i+5] == "false":
			result.WriteString(InspectBoolStyle.Render("false"))
			i += 4
		case c == 'n' && i+4 <= len(jsonStr) && jsonStr[i:i+4] == "null":
			result.WriteString(InspectNullStyle.Render("null"))
			i += 3
		case (c >= '0' && c <= '9') || c == '-' || c == '.':
			// Number - collect all digits
			numStart := i
			for i < len(jsonStr) && ((jsonStr[i] >= '0' && jsonStr[i] <= '9') || jsonStr[i] == '.' || jsonStr[i] == '-' || jsonStr[i] == 'e' || jsonStr[i] == 'E' || jsonStr[i] == '+') {
				i++
			}
			result.WriteString(InspectNumberStyle.Render(jsonStr[numStart:i]))
			continue
		default:
			result.WriteString(string(c))
		}
		i++
	}

	return result.String()
}

// isJSONKey checks if the string starting at position i is a JSON key
func isJSONKey(jsonStr string, startQuote int) bool {
	// Find the closing quote
	i := startQuote + 1
	for i < len(jsonStr) {
		if jsonStr[i] == '"' && (i == 0 || jsonStr[i-1] != '\\') {
			// Found closing quote, now look for colon
			for j := i + 1; j < len(jsonStr); j++ {
				if jsonStr[j] == ':' {
					return true
				}
				if jsonStr[j] != ' ' && jsonStr[j] != '\t' && jsonStr[j] != '\n' && jsonStr[j] != '\r' {
					return false
				}
			}
			return false
		}
		i++
	}
	return false
}

// truncateToWidth truncates a string to fit within maxWidth, properly handling ANSI codes
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}

	currentWidth := lipgloss.Width(s)
	if currentWidth <= maxWidth {
		return s
	}

	// For strings with ANSI codes, we need to be more careful
	// Strip ANSI codes, truncate the plain text, then we lose styling but keep layout correct
	// This is a trade-off: we preserve layout at the cost of styling on truncated lines

	// Use a character-by-character approach that tracks visible width
	targetWidth := maxWidth - 3 // Reserve space for "..."
	visibleWidth := 0
	inEscape := false
	lastSafePos := 0

	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			// Start of ANSI escape sequence
			inEscape = true
			i++
			continue
		}

		if inEscape {
			// Inside escape sequence, look for the terminating character
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			i++
			continue
		}

		// Regular character - count its width
		r, size := decodeRune([]byte(s[i:]))
		charWidth := 1
		if r >= 0x1100 { // Rough check for wide characters (CJK, etc.)
			charWidth = 2
		}

		if visibleWidth+charWidth > targetWidth {
			// We've reached the limit
			break
		}

		visibleWidth += charWidth
		i += size
		lastSafePos = i
	}

	// Find a good truncation point that doesn't break ANSI sequences
	// Append reset code to ensure we don't leave terminal in bad state
	return s[:lastSafePos] + "\x1b[0m..."
}
