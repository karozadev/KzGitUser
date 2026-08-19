package tui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// newHeaderView creates a styled header with title and subtitle.
func newHeaderView(title, subtitle string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignCenter)
	fmt.Fprintf(tv, "[::b]%s\n", title)
	if subtitle != "" {
		fmt.Fprintf(tv, "[#6C7086]%s", subtitle)
	}
	tv.SetBackgroundColor(colorBackground)
	return tv
}

// newSeparator creates a horizontal separator line.
func newSeparator() *tview.TextView {
	tv := tview.NewTextView()
	tv.SetText(strings.Repeat("─", 50))
	tv.SetTextColor(colorBorder)
	tv.SetBackgroundColor(colorBackground)
	return tv
}

// newInfoRow creates a label: value row.
func newInfoRow(label, value string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(colorBackground)
	if value == "" || value == "Not set" {
		fmt.Fprintf(tv, "[#6C7086]%-10s: [colorForeground]%s", label, value)
	} else {
		fmt.Fprintf(tv, "[#6C7086]%-10s: [white]%s", label, value)
	}
	return tv
}

// newStatusBadge creates a colored status badge.
func newStatusBadge(text string, ok bool) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(colorBackground)
	if ok {
		fmt.Fprintf(tv, "[#7EC87E]✓ %s", text)
	} else {
		fmt.Fprintf(tv, "[#E06C75]✗ %s", text)
	}
	return tv
}

// newSectionBox creates a titled box section.
func newSectionBox(title string, content tview.Primitive) *tview.Flex {
	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetBackgroundColor(colorBackground)
	fmt.Fprintf(header, "[#6C9EEB::b] %s ", title)

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.AddItem(header, 1, 0, false)
	flex.AddItem(content, 0, 1, false)
	return flex
}

// newKeyBinding creates a key binding hint text.
func newKeyBinding(key, desc string) string {
	return fmt.Sprintf("[#6C9EEB]%s [white]%s", key, desc)
}

// footerBar creates the bottom footer with key hints.
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
