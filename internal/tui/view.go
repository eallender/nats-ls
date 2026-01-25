// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Build the complete UI
	header := m.renderHeader()
	commandBar := m.renderCommandBar()
	content := m.renderContent()

	// Combine all sections
	if m.commandBarActive {
		return lipgloss.JoinVertical(lipgloss.Left, header, commandBar, content)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

// renderHeader creates the header bar with app info and status
func (m Model) renderHeader() string {
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

	controls := HeaderControlStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"<enter>",
		"<esc>",
		"<↑↓>",
		"<l>",
		"<:>",
		"<q>",
	),
	)

	controlsInfo := HeaderControlStyleInfo.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		"drill down",
		"go back",
		"navigate",
		"logs",
		"filter",
		"quit",
	))

	// Combine logo and status horizontally
	headerContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logo,
		statusInfo,
		controls,
		controlsInfo,
	)

	// Apply container style with padding and width
	return HeaderContainerStyle.
		Width(m.width).
		Padding(0, 1).
		Render(headerContent)
}

// renderContent creates the main content area with nav and info panels
func (m Model) renderContent() string {
	// Calculate widths: Nav = 1/3, Info = fills remaining space
	navWidth := m.width / 3
	infoWidth := m.width - navWidth

	// Build navigation content with hierarchical subjects as a table
	var navText string

	if m.discovery != nil {
		// Add path as a title line if drilled down
		if len(m.navPath) > 0 {
			pathDisplay := strings.Join(m.navPath, ".") + " >"
			// Create a styled title that looks like it's part of the border
			titleLen := len(pathDisplay)
			contentWidth := navWidth - 8 // Account for padding and borders
			leftDashes := (contentWidth - titleLen - 2) / 2
			rightDashes := contentWidth - titleLen - 2 - leftDashes

			titleLine := lipgloss.NewStyle().Foreground(ColorMuted).Render(
				strings.Repeat("─", leftDashes) + " " + pathDisplay + " " + strings.Repeat("─", rightDashes),
			)
			navText = titleLine + "\n\n"
		}

		nodes := m.getSubjectsAtCurrentLevel()
		if len(nodes) > 0 {
			// Table header
			header := NavTableHeaderStyle.Render(
				fmt.Sprintf("%-40s %10s", "SUBJECT", "MESSAGES"),
			)
			navText += header + "\n"

			// Table rows
			for i, node := range nodes {
				rowStyle := NavTableRowStyle
				if i == m.selectedIndex {
					rowStyle = NavTableSelectedRowStyle
				}

				// Display name with indicator for directories vs leaves
				displayName := node.Name
				if !node.IsLeaf {
					displayName += " >"
				}

				// Truncate if too long
				if len(displayName) > 38 {
					displayName = displayName[:35] + "..."
				}

				row := rowStyle.Render(
					fmt.Sprintf("%-40s %10d", displayName, node.MessageCount),
				)
				navText += row + "\n"
			}
		} else {
			navText += "No subjects discovered yet..."
		}
	} else {
		navText = "Not connected..."
	}

	// Navigation panel (1/3 width)
	navContent := NavStyle.
		Width(navWidth).
		Height(m.height - 10).
		Render(navText)

	// Info/main content panel (fills remaining space)
	// Subtract padding (4) and borders (2)
	infoContent := InfoStyle.
		Width(infoWidth - 6).
		Height(m.height - 10).
		Render("Info Panel\n\nContent goes here")

	// Combine navigation and info horizontally
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		navContent,
		infoContent,
	)

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
