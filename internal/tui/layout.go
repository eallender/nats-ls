// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import "github.com/charmbracelet/lipgloss"

// Layout constants
const (
	// Minimum terminal dimensions
	MinTerminalWidth = 80
	MinContentHeight = 5
)

// IsNarrowTerminal returns true if terminal width is below minimum for full layout
func IsNarrowTerminal(width int) bool {
	return width < MinTerminalWidth
}

// GetFrameHeight returns the vertical frame size (padding + borders) for a style
func GetFrameHeight(style lipgloss.Style) int {
	// Get vertical padding
	// GetPadding returns: top, right, bottom, left
	top, _, bottom, _ := style.GetPadding()

	// Get vertical borders
	vBorder := 0
	if style.GetBorderTop() {
		vBorder++
	}
	if style.GetBorderBottom() {
		vBorder++
	}

	return top + bottom + vBorder
}

// MaxContentHeight returns the maximum height available for content
// accounting for the given style's padding and borders
func MaxContentHeight(totalHeight int, style lipgloss.Style) int {
	frameSize := GetFrameHeight(style)
	contentHeight := totalHeight - frameSize

	if contentHeight < 1 {
		contentHeight = 1
	}

	return contentHeight
}

// GetContentWidth calculates the available content width accounting for
// borders (1+1) and padding (2+2) = 6 characters
func GetContentWidth(terminalWidth int) int {
	const framePadding = 6 // 2+2 padding + 1+1 borders
	contentWidth := terminalWidth - framePadding
	if contentWidth < 1 {
		contentWidth = 1
	}
	return contentWidth
}

// EnsureMinimumContentHeight ensures content height meets minimum requirements
// including frame overhead (padding + borders)
func EnsureMinimumContentHeight(contentHeight int, style lipgloss.Style) int {
	frameHeight := GetFrameHeight(style)
	minRequiredHeight := MinContentHeight + frameHeight
	if contentHeight < minRequiredHeight {
		return minRequiredHeight
	}
	return contentHeight
}
