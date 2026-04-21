// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package decode

import (
	"fmt"
	"strings"
)

// HexDecoder formats binary data as a hex dump
// This decoder always succeeds and serves as the fallback
type HexDecoder struct {
	config *DecoderConfig
}

// NewHexDecoder creates a new hex decoder with the given configuration
func NewHexDecoder(config *DecoderConfig) *HexDecoder {
	return &HexDecoder{config: config}
}

// Decode formats data as a hex dump (always succeeds)
func (d *HexDecoder) Decode(data []byte) *DecodeResult {
	formatted := d.formatHexDump(data)

	return &DecodeResult{
		Content:    formatted,
		Format:     "hex",
		Confidence: 1.0, // Always works (fallback)
	}
}

// Name returns the decoder identifier
func (d *HexDecoder) Name() string {
	return "hex"
}

// Priority returns the decoder priority (lowest = 1, fallback)
func (d *HexDecoder) Priority() int {
	return 1
}

// formatHexDump formats binary data as a hex dump
func (d *HexDecoder) formatHexDump(data []byte) string {
	var result strings.Builder
	bytesPerLine := 16

	maxWidth := d.config.MaxWidth
	// Adjust bytes per line based on available width
	// Format: "0000: XX XX XX XX  XX XX XX XX  |........|"
	// Minimum width needed: 6 (offset) + 3*8 + 2 (gap) + 3*8 + 2 (gap) + 10 (ascii) = ~70
	if maxWidth < 70 {
		bytesPerLine = 8
	}

	result.WriteString(d.config.Styles.Header.Render("Binary data - hex dump:") + "\n\n")

	for offset := 0; offset < len(data); offset += bytesPerLine {
		// Offset
		result.WriteString(d.config.Styles.Number.Render(fmt.Sprintf("%04x", offset)))
		result.WriteString(d.config.Styles.Bracket.Render(": "))

		// Hex bytes
		end := offset + bytesPerLine
		if end > len(data) {
			end = len(data)
		}

		for i := offset; i < offset+bytesPerLine; i++ {
			if i < end {
				result.WriteString(d.config.Styles.Key.Render(fmt.Sprintf("%02x ", data[i])))
			} else {
				result.WriteString("   ")
			}
			if i == offset+bytesPerLine/2-1 {
				result.WriteString(" ")
			}
		}

		// ASCII representation
		result.WriteString(d.config.Styles.Bracket.Render(" |"))
		for i := offset; i < end; i++ {
			if data[i] >= 32 && data[i] < 127 {
				result.WriteString(d.config.Styles.String.Render(string(data[i])))
			} else {
				result.WriteString(d.config.Styles.Null.Render("."))
			}
		}
		result.WriteString(d.config.Styles.Bracket.Render("|"))

		if offset+bytesPerLine < len(data) {
			result.WriteString("\n")
		}
	}

	return result.String()
}
