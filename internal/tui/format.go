// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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
