// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderMessageInspectWithHeight creates the message inspector view with scrollable pretty JSON
func (m Model) renderMessageInspectWithHeight(contentHeight int) string {
	// Enforce minimum content height
	contentHeight = EnsureMinimumContentHeight(contentHeight, InspectViewStyle)

	// Calculate content dimensions
	contentWidth := GetContentWidth(m.width)
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

			// Format the message data using the decoder registry
			m.decoderRegistry.UpdateWidth(contentWidth)
			result := m.decoderRegistry.DecodeWithFallback(msg.Data)
			formattedData := result.Content

			// Split into lines for scrolling
			allLines := strings.Split(header+formattedData, "\n")

			// Calculate available lines for content (minus navigation hint)
			availableLines := contentHeightAdjusted - 1
			availableLines = max(availableLines, 1)

			// Calculate max scroll (clamping is done in Update, not here)
			maxScroll := len(allLines) - availableLines
			maxScroll = max(maxScroll, 0)

			// Clamp scroll offset for display (read-only)
			scrollOffset := m.inspectScrollOffset
			scrollOffset = min(scrollOffset, maxScroll)

			// Get visible lines
			endIdx := scrollOffset + availableLines
			endIdx = min(endIdx, len(allLines))

			visibleLines := allLines[scrollOffset:endIdx]

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

			// Check if paused
			isPaused := m.viewer != nil && m.viewer.IsPaused()

			// Add scroll indicator
			var scrollInfo string
			if isPaused {
				if maxScroll == 0 {
					scrollInfo = fmt.Sprintf("── PAUSED │ message %d of %d ──", m.inspectedMessageIndex+1, len(messages))
				} else {
					scrollInfo = fmt.Sprintf("── PAUSED │ message %d of %d │ line %d-%d of %d ──",
						m.inspectedMessageIndex+1, len(messages),
						scrollOffset+1, endIdx, len(allLines))
				}
			} else if maxScroll == 0 {
				scrollInfo = fmt.Sprintf("── message %d of %d ──", m.inspectedMessageIndex+1, len(messages))
			} else {
				scrollInfo = fmt.Sprintf("── message %d of %d │ line %d-%d of %d ──",
					m.inspectedMessageIndex+1, len(messages),
					scrollOffset+1, endIdx, len(allLines))
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

	styledTitle := BorderTitleStyle.Render(borderTitle)
	content = insertBorderTitle(content, styledTitle, contentWidth+6)

	return content
}
