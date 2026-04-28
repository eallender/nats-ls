// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package decode

import "github.com/charmbracelet/lipgloss"

// DecoderConfig holds configuration for message decoders
type DecoderConfig struct {
	MaxWidth int         // Terminal width for wrapping
	Styles   StyleConfig // Lipgloss styles for syntax highlighting
}

// StyleConfig holds lipgloss styles for syntax highlighting
type StyleConfig struct {
	Key     lipgloss.Style // JSON keys, hex bytes
	String  lipgloss.Style // String values, printable ASCII
	Number  lipgloss.Style // Numeric values, hex offsets
	Bool    lipgloss.Style // Boolean values
	Null    lipgloss.Style // Null values, non-printable chars
	Bracket lipgloss.Style // Brackets, delimiters
	Header  lipgloss.Style // Section headers
}

// decodeRune decodes a UTF-8 rune from bytes
// Returns the decoded rune and the number of bytes consumed
// Returns U+FFFD (replacement character) if the bytes are invalid UTF-8
func decodeRune(data []byte) (rune, int) {
	if len(data) == 0 {
		return '\uFFFD', 0
	}

	// Simple UTF-8 decoding
	b := data[0]
	if b < 0x80 {
		// Single-byte ASCII character
		return rune(b), 1
	}
	if b < 0xC0 {
		// Invalid UTF-8 start byte
		return '\uFFFD', 1
	}
	if b < 0xE0 && len(data) >= 2 {
		// Two-byte character
		return rune(b&0x1F)<<6 | rune(data[1]&0x3F), 2
	}
	if b < 0xF0 && len(data) >= 3 {
		// Three-byte character
		return rune(b&0x0F)<<12 | rune(data[1]&0x3F)<<6 | rune(data[2]&0x3F), 3
	}
	if len(data) >= 4 {
		// Four-byte character
		return rune(b&0x07)<<18 | rune(data[1]&0x3F)<<12 | rune(data[2]&0x3F)<<6 | rune(data[3]&0x3F), 4
	}
	return '\uFFFD', 1
}
