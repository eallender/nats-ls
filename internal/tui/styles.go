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
	HeaderContainerStyle = lipgloss.NewStyle()

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
	BorderTitleStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	NavStyle = lipgloss.NewStyle().
			PaddingTop(0).
			PaddingBottom(0).
			PaddingLeft(2).
			PaddingRight(2).
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

// Command bar styles
var (
	CommandBarStyle = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Background(ColorBackground).
		Padding(0, 1)
)

// Log view styles
var (
	LogViewStyle = lipgloss.NewStyle().
			Padding(1, 2).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorMuted)

	LogTimestampStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	LogSubjectStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	LogDataStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	LogEmptyStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	LogSelectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(ColorPrimary).
				Bold(true)
)

// Message inspector styles
var (
	InspectViewStyle = lipgloss.NewStyle().
				Padding(1, 2).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorMuted)

	InspectKeyStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	InspectStringStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	InspectNumberStyle = lipgloss.NewStyle().
				Foreground(ColorWarning)

	InspectBoolStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	InspectNullStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true)

	InspectBracketStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	InspectHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
)
