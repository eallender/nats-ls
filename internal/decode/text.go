// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package decode

import "strings"

// TextDecoder decodes printable UTF-8 text
type TextDecoder struct {
	config *DecoderConfig
}

// NewTextDecoder creates a new text decoder with the given configuration
func NewTextDecoder(config *DecoderConfig) *TextDecoder {
	return &TextDecoder{config: config}
}

// Decode attempts to decode data as printable UTF-8 text
// Returns nil if less than 90% of the data is printable
func (d *TextDecoder) Decode(data []byte) *DecodeResult {
	if !d.isPrintableText(data) {
		return nil // Not printable text
	}

	text := sanitizeControlChars(string(data))
	wrapped := d.wrapText(text)

	confidence := d.calculateConfidence(data)

	return &DecodeResult{
		Content:    wrapped,
		Format:     "text",
		Confidence: confidence,
	}
}

// Name returns the decoder identifier
func (d *TextDecoder) Name() string {
	return "text"
}

// Priority returns the decoder priority (medium = 5)
func (d *TextDecoder) Priority() int {
	return 5
}

// isPrintableText checks if data is mostly printable UTF-8 text
// Uses a 90% threshold to determine if data should be treated as text
func (d *TextDecoder) isPrintableText(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	printable := 0
	total := 0

	for i := 0; i < len(data); {
		var r rune
		var size int
		if data[i] >= 0x80 {
			// Multi-byte UTF-8
			r, size = decodeRune(data[i:])
			if r == '\uFFFD' {
				// Invalid UTF-8
				total++
				i++
				continue
			}
		} else {
			r = rune(data[i])
			size = 1
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

// calculateConfidence returns the ratio of printable characters
func (d *TextDecoder) calculateConfidence(data []byte) float64 {
	if len(data) == 0 {
		return 1.0
	}

	printable := 0
	total := 0

	for i := 0; i < len(data); {
		var r rune
		var size int
		if data[i] >= 0x80 {
			r, size = decodeRune(data[i:])
		} else {
			r = rune(data[i])
			size = 1
		}

		total++
		if (r >= 32 && r < 127) || r == '\n' || r == '\r' || r == '\t' || (r >= 0x80 && r != '\uFFFD') {
			printable++
		}
		i += size
	}

	if total == 0 {
		return 1.0
	}
	return float64(printable) / float64(total)
}

// sanitizeControlChars replaces non-printable control characters with '·' to
// prevent stray bytes (e.g. \x1b) from corrupting terminal rendering.
func sanitizeControlChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			result.WriteRune('·')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// wrapText wraps text at maxWidth characters
func (d *TextDecoder) wrapText(text string) string {
	maxWidth := d.config.MaxWidth
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
