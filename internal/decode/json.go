// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package decode

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// JSONDecoder decodes and syntax-highlights JSON data
type JSONDecoder struct {
	config *DecoderConfig
}

// NewJSONDecoder creates a new JSON decoder with the given configuration
func NewJSONDecoder(config *DecoderConfig) *JSONDecoder {
	return &JSONDecoder{config: config}
}

// Decode attempts to decode data as JSON
// Returns nil if data is not valid JSON
func (d *JSONDecoder) Decode(data []byte) *DecodeResult {
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil // Not valid JSON
	}

	// Pretty print the JSON
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(jsonData); err != nil {
		return nil
	}

	formatted := strings.TrimSpace(buf.String())
	highlighted := d.syntaxHighlight(formatted)

	// Wrap lines to fit width
	wrapped := d.wrapLines(highlighted)

	return &DecodeResult{
		Content:    wrapped,
		Format:     "json",
		Confidence: 1.0,
		Metadata:   map[string]any{"valid_json": true},
	}
}

// Name returns the decoder identifier
func (d *JSONDecoder) Name() string {
	return "json"
}

// Priority returns the decoder priority (highest = 10)
func (d *JSONDecoder) Priority() int {
	return 10
}

// syntaxHighlight applies syntax highlighting to formatted JSON
func (d *JSONDecoder) syntaxHighlight(jsonStr string) string {
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
					result.WriteString(d.config.Styles.Key.Render("\""))
				} else {
					result.WriteString(d.config.Styles.String.Render("\""))
				}
				inString = false
				inKey = false
			} else {
				// Start of string - check if it's a key
				inString = true
				// Look ahead to see if this is a key (followed by : after closing quote)
				inKey = d.isJSONKey(jsonStr, i)
				if inKey {
					result.WriteString(d.config.Styles.Key.Render("\""))
				} else {
					result.WriteString(d.config.Styles.String.Render("\""))
				}
			}
		case inString:
			// Inside a string, continue with current style
			if inKey {
				result.WriteString(d.config.Styles.Key.Render(string(c)))
			} else {
				result.WriteString(d.config.Styles.String.Render(string(c)))
			}
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ':' || c == ',':
			result.WriteString(d.config.Styles.Bracket.Render(string(c)))
		case c == 't' && i+4 <= len(jsonStr) && jsonStr[i:i+4] == "true":
			result.WriteString(d.config.Styles.Bool.Render("true"))
			i += 3
		case c == 'f' && i+5 <= len(jsonStr) && jsonStr[i:i+5] == "false":
			result.WriteString(d.config.Styles.Bool.Render("false"))
			i += 4
		case c == 'n' && i+4 <= len(jsonStr) && jsonStr[i:i+4] == "null":
			result.WriteString(d.config.Styles.Null.Render("null"))
			i += 3
		case (c >= '0' && c <= '9') || c == '-' || c == '.':
			// Number - collect all digits
			numStart := i
			for i < len(jsonStr) && ((jsonStr[i] >= '0' && jsonStr[i] <= '9') || jsonStr[i] == '.' || jsonStr[i] == '-' || jsonStr[i] == 'e' || jsonStr[i] == 'E' || jsonStr[i] == '+') {
				i++
			}
			result.WriteString(d.config.Styles.Number.Render(jsonStr[numStart:i]))
			continue
		default:
			result.WriteString(string(c))
		}
		i++
	}

	return result.String()
}

// isJSONKey checks if the string starting at position i is a JSON key
func (d *JSONDecoder) isJSONKey(jsonStr string, startQuote int) bool {
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

// wrapLines wraps lines to fit within maxWidth, handling ANSI codes
func (d *JSONDecoder) wrapLines(text string) string {
	maxWidth := d.config.MaxWidth
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
		} else {
			// Line too long - for JSON we just keep it as is
			// The TUI will handle truncation if needed
			result.WriteString(line)
		}
	}

	return result.String()
}
