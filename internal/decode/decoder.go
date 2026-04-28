// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package decode

import "sort"

// DecodeResult represents the output of a decoder
type DecodeResult struct {
	Content    string         // Styled content with ANSI codes
	Format     string         // Format identifier ("json", "text", "hex")
	Confidence float64        // Confidence score (0.0-1.0)
	Metadata   map[string]any // Optional metadata about the decode
}

// Decoder interface defines how to decode message data
// Implement this interface to add a new decoder
type Decoder interface {
	// Decode attempts to decode data, returns nil if unable to decode
	Decode(data []byte) *DecodeResult

	// Name returns the decoder's unique identifier
	Name() string

	// Priority returns the sort order (higher priority = tried first)
	Priority() int
}

// Registry manages a collection of decoders
type Registry struct {
	decoders []Decoder
	config   *DecoderConfig
}

// NewRegistry creates a new decoder registry with the given configuration
func NewRegistry(config *DecoderConfig) *Registry {
	return &Registry{
		decoders: make([]Decoder, 0),
		config:   config,
	}
}

// Register adds a decoder to the registry
// Decoders are automatically sorted by priority (highest first)
func (r *Registry) Register(decoder Decoder) {
	r.decoders = append(r.decoders, decoder)

	// Sort by priority (descending - highest priority first)
	sort.Slice(r.decoders, func(i, j int) bool {
		return r.decoders[i].Priority() > r.decoders[j].Priority()
	})
}

// DecodeWithFallback tries decoders in priority order until one succeeds
// Returns the result from the first decoder that successfully decodes the data
func (r *Registry) DecodeWithFallback(data []byte) DecodeResult {
	for _, decoder := range r.decoders {
		if result := decoder.Decode(data); result != nil {
			return *result
		}
	}

	return DecodeResult{
		Content: "Unable to decode message: binary or unknown format",
		Format:  "unknown",
	}
}

// UpdateWidth updates the maximum width for wrapping
func (r *Registry) UpdateWidth(width int) {
	r.config.MaxWidth = width
}

// Config returns a pointer to the registry's configuration
// This allows decoders to access dynamic config changes
func (r *Registry) Config() *DecoderConfig {
	return r.config
}
