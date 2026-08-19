package tui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

func newHeaderView(title, subtitle string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignCenter)
	_, _ = fmt.Fprintf(tv, "[::b]%s\n", title)
	if subtitle != "" {
		_, _ = fmt.Fprintf(tv, "[#6C7086]%s", subtitle)
	}
	tv.SetBackgroundColor(colorBackground)
	return tv
}

func newSeparator() *tview.TextView {
	tv := tview.NewTextView()
	tv.SetText(strings.Repeat("─", 50))
	tv.SetTextColor(colorBorder)
	tv.SetBackgroundColor(colorBackground)
	return tv
}

func newInfoRow(label, value string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(colorBackground)
	if value == "" || value == "Not set" {
		_, _ = fmt.Fprintf(tv, "[#6C7086]%-10s: [colorForeground]%s", label, value)
	} else {
		_, _ = fmt.Fprintf(tv, "[#6C7086]%-10s: [white]%s", label, value)
	}
	return tv
}

func newKeyBinding(key, desc string) string {
	return fmt.Sprintf("[#6C9EEB]%s [white]%s", key, desc)
}

func newFooterBar() *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(colorBackground)
	hints := []string{
		newKeyBinding("Tab", "Next"),
		newKeyBinding("/", "Command"),
		newKeyBinding("?", "Help"),
		newKeyBinding("q", "Quit"),
	}
	tv.SetText("  " + strings.Join(hints, "  │  "))
	return tv
}
