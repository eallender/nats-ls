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

// renderContentWithHeight creates the main content area with nav and info panels
func (m Model) renderContentWithHeight(contentHeight int) string {
	// Create layout helper
	layout := NewLayout(m.width, m.height)

	// Enforce minimum content height (must account for frame overhead)
	// The content boxes need frame space (padding+borders) plus some content
	frameHeight := GetFrameHeight(NavStyle) // NavStyle and InfoStyle have same frame
	minRequiredHeight := MinContentHeight + frameHeight
	if contentHeight < minRequiredHeight {
		contentHeight = minRequiredHeight
	}

	// Split width using percentage ratio (33% nav, 67% info)
	navWidth, infoWidth := layout.SplitHorizontal(NavWidthRatio)

	// Calculate content widths (accounting for padding and borders)
	navContentWidth := MaxContentWidth(navWidth, NavStyle)
	infoContentWidth := MaxContentWidth(infoWidth, InfoStyle)

	// Build navigation content with hierarchical subjects as a table
	var navText string

	if m.discovery != nil {
		// Add path as a title line if drilled down
		if len(m.navPath) > 0 {
			pathDisplay := strings.Join(m.navPath, ".") + " >"
			// Create a styled title that looks like it's part of the border
			titleLen := len(pathDisplay)

			leftDashes := (navContentWidth - titleLen - 2) / 2
			if leftDashes < 0 {
				leftDashes = 0
			}
			rightDashes := navContentWidth - titleLen - 2 - leftDashes
			if rightDashes < 0 {
				rightDashes = 0
			}

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

	// Calculate inner content heights (accounting for padding and borders dynamically)
	navContentHeight := MaxContentHeight(contentHeight, NavStyle)
	infoContentHeight := MaxContentHeight(contentHeight, InfoStyle)

	// Navigation panel - use explicit Width and Height for proper sizing
	navContent := NavStyle.
		Width(navContentWidth).
		Height(navContentHeight).
		Render(navText)

	// Info panel - use explicit Width and Height for proper sizing
	infoContent := InfoStyle.
		Width(infoContentWidth).
		Height(infoContentHeight).
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
