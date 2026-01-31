// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
