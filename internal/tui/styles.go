// Package tui implements the Terminal User Interface for KzGitUser.
package tui

import "github.com/gdamore/tcell/v2"

var (
	colorPrimary    = tcell.NewHexColor(0x6C9EEB) // Blue
	colorSuccess    = tcell.NewHexColor(0x7EC87E) // Green
	colorError      = tcell.NewHexColor(0xE06C75) // Red
	colorWarning    = tcell.NewHexColor(0xE5C07B) // Yellow
	colorMuted      = tcell.NewHexColor(0x6C7086) // Gray
	colorForeground = tcell.NewHexColor(0xCDD6F4) // Light
	colorBackground = tcell.NewHexColor(0x1E1E2E) // Dark
	colorBorder     = tcell.NewHexColor(0x45475A) // Border gray
	colorHighlight  = tcell.NewHexColor(0x313244) // Selection bg
	colorAccent     = tcell.NewHexColor(0xCBA6F7) // Purple accent
)

var (
	styleTitle    = tcell.StyleDefault.Foreground(colorPrimary).Bold(true)
	styleLabel    = tcell.StyleDefault.Foreground(colorMuted)
	styleValue    = tcell.StyleDefault.Foreground(colorForeground)
	styleSuccess  = tcell.StyleDefault.Foreground(colorSuccess)
	styleError    = tcell.StyleDefault.Foreground(colorError)
	styleWarning  = tcell.StyleDefault.Foreground(colorWarning)
	styleMuted    = tcell.StyleDefault.Foreground(colorMuted)
	styleBorder   = tcell.StyleDefault.Foreground(colorBorder)
	styleAccent   = tcell.StyleDefault.Foreground(colorAccent).Bold(true)
	styleHeader   = tcell.StyleDefault.Foreground(colorPrimary).Bold(true)
	styleDim      = tcell.StyleDefault.Foreground(colorMuted)
	styleSelected = tcell.StyleDefault.Foreground(colorBackground).Background(colorPrimary).Bold(true)
)

const (
	appTitle    = " KzGitUser "
	appSubtitle = " Git Identity Manager "
)
