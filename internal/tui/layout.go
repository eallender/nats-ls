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

// Layout provides helpers for responsive TUI layout calculations
type Layout struct {
	TerminalWidth  int
	TerminalHeight int
}

// NewLayout creates a layout helper for the given terminal dimensions
func NewLayout(width, height int) Layout {
	return Layout{
		TerminalWidth:  width,
		TerminalHeight: height,
	}
}

// IsNarrow returns true if terminal width is below minimum for full layout
func (l Layout) IsNarrow() bool {
	return l.TerminalWidth < MinTerminalWidth
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
