// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"
)

// renderContentWithHeight creates the main content area with a single full-width panel
func (m Model) renderContentWithHeight(contentHeight int) string {
	// Calculate content width and the actual content area (excluding frame)
	// Note: Don't use EnsureMinimumContentHeight - trust the exact space from View()
	contentWidth := GetContentWidth(m.width)
	// MaxContentHeight returns space available inside the box (minus padding/borders)
	innerContentHeight := MaxContentHeight(contentHeight, NavStyle)

	// Build main content with hierarchical subjects as a table
	var mainText string

	// Determine border title if drilled down or filter is active
	var borderTitle string
	if m.discovery != nil {
		if len(m.navPath) > 0 {
			borderTitle = strings.Join(m.navPath, ".") + ".>"
		}
		// Add filter indicator
		if m.subjectFilter != "" {
			if borderTitle != "" {
				borderTitle += " "
			}
			borderTitle += fmt.Sprintf("[filter: %s]", m.subjectFilter)
		}
		// Truncate if too long
		maxTitleLen := contentWidth - 4
		if len(borderTitle) > maxTitleLen && maxTitleLen > 3 {
			borderTitle = borderTitle[:maxTitleLen-3] + "..."
		}
	}

	if m.discovery != nil {
		nodes := m.getSubjectsAtCurrentLevel()
		if len(nodes) > 0 {
			// Calculate column widths dynamically based on available space
			var msgColWidth, rateColWidth, ageColWidth, idleColWidth, subjectColWidth int
			spacingChars := 12 // spaces between columns (3 spaces between each pair)

			// Scale columns based on available width
			if contentWidth < 60 {
				// Very narrow terminal - use minimal widths
				msgColWidth = 6
				rateColWidth = 7
				ageColWidth = 6
				idleColWidth = 9
				subjectColWidth = contentWidth - msgColWidth - rateColWidth - ageColWidth - idleColWidth - spacingChars
				if subjectColWidth < 8 {
					subjectColWidth = 8
				}
			} else {
				// Normal width - use comfortable column sizes
				msgColWidth = 10
				rateColWidth = 9
				ageColWidth = 10
				idleColWidth = 10
				subjectColWidth = contentWidth - msgColWidth - rateColWidth - ageColWidth - idleColWidth - spacingChars

				// Ensure min safe column width
				subjectColWidth = max(subjectColWidth, 15)
			}

			// Final safety check: ensure total width doesn't exceed contentWidth
			totalWidth := subjectColWidth + msgColWidth + rateColWidth + ageColWidth + idleColWidth + spacingChars
			if totalWidth > contentWidth {
				// Force subjectColWidth to fit within bounds
				subjectColWidth = contentWidth - msgColWidth - rateColWidth - ageColWidth - idleColWidth - spacingChars
				subjectColWidth = max(subjectColWidth, 1)
			}

			// Calculate visible range of subjects
			totalNodes := len(nodes)

			// Reserve space: 1 line for header, 1 line for potential scroll indicator
			// Always reserve space for scroll indicator to prevent layout shifts
			availableLines := innerContentHeight - 2
			if availableLines < 1 {
				availableLines = 1
			}

			// Determine if we need to show scroll indicator
			needsScrollIndicator := totalNodes > availableLines

			startIdx := m.browseScrollOffset
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx >= totalNodes {
				startIdx = totalNodes - 1
				if startIdx < 0 {
					startIdx = 0
				}
			}
			endIdx := startIdx + availableLines
			if endIdx > totalNodes {
				endIdx = totalNodes
			}

			// Table header with dynamic column widths
			headerText := fmt.Sprintf("%-*s   %-*s   %-*s   %-*s   %-*s", subjectColWidth, "SUBJECT", msgColWidth, "MESSAGES", rateColWidth, "RATE", ageColWidth, "AGE", idleColWidth, "LAST SEEN")
			// Ensure exact width to prevent wrapping
			headerText = ensureWidth(headerText, contentWidth)
			header := NavTableHeaderStyle.Render(headerText)
			mainText += header + "\n"

			// Table rows (only render visible ones)
			for i := startIdx; i < endIdx; i++ {
				node := nodes[i]
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

				// Format the new columns
				rateStr := formatRate(node.MessageCount, node.FirstSeen)
				ageStr := formatRelativeTime(node.FirstSeen)
				idleStr := formatIdleTime(node.LastSeen)

				rowText := fmt.Sprintf("%-*s   %-*d   %-*s   %-*s   %-*s", subjectColWidth, displayName, msgColWidth, node.MessageCount, rateColWidth, rateStr, ageColWidth, ageStr, idleColWidth, idleStr)
				// Ensure exact width to prevent wrapping
				rowText = ensureWidth(rowText, contentWidth)
				row := rowStyle.Render(rowText)
				mainText += row + "\n"
			}

			// Add scroll indicator if needed (last row's \n naturally separates it)
			if needsScrollIndicator {
				scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, totalNodes)
				mainText += NavTableHeaderStyle.Render(ensureWidth(scrollInfo, contentWidth))
			} else {
				// Add empty line to maintain consistent height
				mainText += NavTableHeaderStyle.Render(ensureWidth("", contentWidth))
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
	// Use Height with contentHeight to fill the allocated space exactly
	boxStyle := NavStyle.Height(contentHeight)

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
