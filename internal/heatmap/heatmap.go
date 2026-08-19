// Package heatmap renders a GitHub-style commit contribution heatmap as
// ANSI-colored terminal text. It knows nothing about Git or profiles — it
// only turns a date-to-count map into lines of text.
package heatmap

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// weekdayLabels are the row labels, Monday through Sunday.
var weekdayLabels = [7]string{"Lun", "Mar", "Mer", "Jeu", "Ven", "Sam", "Dim"}

// levelChars are the block characters for intensity levels 0 (no activity)
// through 4 (busiest).
var levelChars = [5]rune{'░', '░', '▒', '▓', '█'}

// levelColors are ANSI 256-color codes approximating GitHub's green
// contribution scale, from "no activity" (dim gray) to "very active".
var levelColors = [5]string{
	"\x1b[38;5;238m",
	"\x1b[38;5;22m",
	"\x1b[38;5;28m",
	"\x1b[38;5;34m",
	"\x1b[38;5;46m",
}

const colorReset = "\x1b[0m"

// Options configures a single profile's heatmap.
type Options struct {
	// ProfileName and Email identify whose contributions this is; both
	// are display-only.
	ProfileName string
	Email       string
	// Days is the size of the trailing window to render, ending today.
	Days int
	// Counts maps "2006-01-02" dates to commit counts.
	Counts map[string]int
	// NoColor disables ANSI color codes, e.g. for non-terminal output.
	NoColor bool
}

// Render returns a profile's heatmap as a slice of lines, with no trailing
// newline on the last line — ready to print directly, or to combine with
// other profiles' output via SideBySide.
func Render(opts Options) []string {
	days := opts.Days
	if days <= 0 {
		days = 30
	}

	today := truncateToDay(time.Now())
	start := today.AddDate(0, 0, -(days - 1))
	// Align the grid's first column back to the most recent Monday on or
	// before start, so every column is a full Mon-Sun week.
	weekdayIdx := (int(start.Weekday()) + 6) % 7 // Monday=0 .. Sunday=6
	gridStart := start.AddDate(0, 0, -weekdayIdx)

	total, max := summarize(opts.Counts, start, today)

	lines := []string{
		fmt.Sprintf("Profile: %s <%s>", opts.ProfileName, opts.Email),
		fmt.Sprintf("Period:  last %d days (%s -> %s)", days, start.Format("2006-01-02"), today.Format("2006-01-02")),
		fmt.Sprintf("Total:   %d commits", total),
		"",
	}

	grid := buildGrid(opts.Counts, gridStart, start, today, max, opts.NoColor)
	for r := 0; r < 7; r++ {
		lines = append(lines, weekdayLabels[r]+" "+strings.Join(grid[r], " "))
	}

	lines = append(lines, "", legendLine(opts.NoColor))
	return lines
}

// summarize totals only the counts that actually fall within [start, end],
// and finds the highest single-day count in that window (used to scale
// intensity levels relative to this profile's own activity).
func summarize(counts map[string]int, start, end time.Time) (total, max int) {
	for d, c := range counts {
		t, err := time.Parse("2006-01-02", d)
		if err != nil || t.Before(start) || t.After(end) {
			continue
		}
		total += c
		if c > max {
			max = c
		}
	}
	return total, max
}

func buildGrid(counts map[string]int, gridStart, start, end time.Time, max int, noColor bool) [7][]string {
	numDays := int(end.Sub(gridStart).Hours()/24) + 1
	numWeeks := (numDays + 6) / 7

	var grid [7][]string
	for r := range grid {
		grid[r] = make([]string, numWeeks)
	}

	for w := 0; w < numWeeks; w++ {
		for r := 0; r < 7; r++ {
			date := gridStart.AddDate(0, 0, w*7+r)
			if date.Before(start) || date.After(end) {
				grid[r][w] = "  "
				continue
			}
			count := counts[date.Format("2006-01-02")]
			grid[r][w] = cell(intensityLevel(count, max), noColor)
		}
	}
	return grid
}

func cell(level int, noColor bool) string {
	block := string(levelChars[level]) + string(levelChars[level])
	if noColor {
		return block
	}
	return levelColors[level] + block + colorReset
}

func legendLine(noColor bool) string {
	var b strings.Builder
	b.WriteString("Less ")
	for lvl := 0; lvl < 5; lvl++ {
		char := string(levelChars[lvl])
		if noColor {
			b.WriteString(char)
		} else {
			b.WriteString(levelColors[lvl] + char + colorReset)
		}
		b.WriteString(" ")
	}
	b.WriteString("More")
	return b.String()
}

// intensityLevel maps a day's commit count to a level from 0 (none) to 4
// (busiest), scaled relative to max — the highest single-day count in the
// window — rather than fixed thresholds, so the scale adapts to each
// profile's own activity instead of one person's "busy" being another's
// "quiet".
func intensityLevel(count, max int) int {
	if count == 0 {
		return 0
	}
	if max <= 1 {
		return 4
	}
	level := 1 + (count-1)*3/(max-1)
	if level > 4 {
		level = 4
	}
	return level
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// visibleWidth measures a line's printable width, ignoring ANSI color
// escape sequences (which occupy zero terminal columns).
func visibleWidth(s string) int {
	return len([]rune(ansiEscape.ReplaceAllString(s, "")))
}

// SideBySide joins multiple rendered blocks horizontally, line by line,
// separated by gap. Shorter blocks are padded with blank lines, and each
// line is right-padded with plain spaces (accounting for invisible ANSI
// codes) so columns from different blocks stay aligned regardless of
// differing header/legend line lengths.
func SideBySide(blocks [][]string, gap string) []string {
	if len(blocks) == 0 {
		return nil
	}

	maxLines := 0
	widths := make([]int, len(blocks))
	for i, b := range blocks {
		if len(b) > maxLines {
			maxLines = len(b)
		}
		for _, line := range b {
			if w := visibleWidth(line); w > widths[i] {
				widths[i] = w
			}
		}
	}

	out := make([]string, 0, maxLines)
	for row := 0; row < maxLines; row++ {
		parts := make([]string, len(blocks))
		for i, b := range blocks {
			line := ""
			if row < len(b) {
				line = b[row]
			}
			pad := widths[i] - visibleWidth(line)
			if pad < 0 {
				pad = 0
			}
			parts[i] = line + strings.Repeat(" ", pad)
		}
		out = append(out, strings.Join(parts, gap))
	}
	return out
}
