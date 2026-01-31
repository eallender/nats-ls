// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
