// Package tui implements the Terminal User Interface for KzGitUser.
package tui

import "github.com/gdamore/tcell/v2"

var (
	colorPrimary    = tcell.NewHexColor(0x6C9EEB)
	colorError      = tcell.NewHexColor(0xE06C75)
	colorMuted      = tcell.NewHexColor(0x6C7086)
	colorForeground = tcell.NewHexColor(0xCDD6F4)
	colorBackground = tcell.NewHexColor(0x1E1E2E)
	colorBorder     = tcell.NewHexColor(0x45475A)
	colorHighlight  = tcell.NewHexColor(0x313244)
)
