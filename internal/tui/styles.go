// SPDX-License-Identifier: Apache-2.0
// Copyright Evan Allender

package tui

import "github.com/charmbracelet/lipgloss"

// ASCII art logo
const Logo = `   _  ____   ____
  / |/ / /  / __/
 /    / /___\ \  
/_/|_/____/___/  `

// Color palette
var (
	ColorPrimary    = lipgloss.Color("205") // pink/magenta
	ColorSuccess    = lipgloss.Color("42")  // green
	ColorError      = lipgloss.Color("196") // red
	ColorWarning    = lipgloss.Color("220") // yellow
	ColorInfo       = lipgloss.Color("99")  // purple
	ColorMuted      = lipgloss.Color("240") // gray
	ColorBackground = lipgloss.Color("235") // dark gray
)

// Header styles
var (
	HeaderContainerStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true)

	HeaderAppNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Padding(0, 1).
				MarginRight(2)

	HeaderConnectedStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Padding(0, 1)

	HeaderDisconnectedStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Padding(0, 1)

	HeaderServerStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)

	HeaderStatsStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)

	HeaderDividerStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	HeaderControlStyle = lipgloss.NewStyle().
				Foreground(ColorInfo).
				MarginRight(1)

	HeaderControlStyleInfo = lipgloss.NewStyle().
				Foreground(ColorMuted)

	HeaderStatusInfoStyle = lipgloss.NewStyle().
				MarginRight(6)
)

// Navigation styles
var (
	NavStyle = lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorMuted)

	NavTableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	NavTableRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	NavTableSelectedRowStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("0")).
					Background(ColorPrimary).
					Bold(true)
)

// Info styles
var (
	InfoStyle = lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorMuted)
)

// Command bar styles
var (
	CommandBarStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Background(ColorBackground).
		Padding(0, 1)
)
