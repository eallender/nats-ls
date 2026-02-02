// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"
)

// renderContentWithHeight creates the main content area with a single full-width panel
func (m Model) renderContentWithHeight(contentHeight int) string {
	// Enforce minimum content height
	contentHeight = EnsureMinimumContentHeight(contentHeight, NavStyle)

	// Calculate content width and height
	contentWidth := GetContentWidth(m.width)
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
		styledTitle := BorderTitleStyle.Render(borderTitle)
		boxStyle = boxStyle.BorderTop(true).BorderBottom(true).BorderLeft(true).BorderRight(true)
		content := boxStyle.Render(mainText)
		// Insert title into top border
		// Border width = content + horizontal padding (4) + borders (2) = contentWidth + 6
		content = insertBorderTitle(content, styledTitle, contentWidth+6)
		return content
	}

	return boxStyle.Render(mainText)
}
