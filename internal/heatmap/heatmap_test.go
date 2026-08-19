package heatmap

import (
	"strings"
	"testing"
	"time"
)

func TestRender_HeaderContainsMetadata(t *testing.T) {
	lines := Render(Options{
		ProfileName: "work",
		Email:       "john@company.com",
		Days:        30,
		Counts:      map[string]int{},
		NoColor:     true,
	})

	joined := strings.Join(lines, "\n")
	for _, want := range []string{"work", "john@company.com", "last 30 days", "Total:   0 commits"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, joined)
		}
	}
}

func TestRender_TotalOnlyCountsWithinWindow(t *testing.T) {
	today := truncateToDay(time.Now())
	inWindow := today.AddDate(0, 0, -1).Format("2006-01-02")
	outOfWindow := today.AddDate(0, 0, -100).Format("2006-01-02")

	lines := Render(Options{
		ProfileName: "work",
		Days:        7,
		Counts:      map[string]int{inWindow: 3, outOfWindow: 99},
		NoColor:     true,
	})

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Total:   3 commits") {
		t.Fatalf("expected total to exclude out-of-window commits, got:\n%s", joined)
	}
}

func TestRender_GridHasSevenWeekdayRows(t *testing.T) {
	lines := Render(Options{ProfileName: "work", Days: 30, Counts: map[string]int{}, NoColor: true})

	found := 0
	for _, label := range weekdayLabels {
		for _, line := range lines {
			if strings.HasPrefix(line, label+" ") {
				found++
				break
			}
		}
	}
	if found != 7 {
		t.Fatalf("expected all 7 weekday rows (%v), found %d in:\n%s", weekdayLabels, found, strings.Join(lines, "\n"))
	}
}

func TestRender_NoColorOmitsEscapeCodes(t *testing.T) {
	lines := Render(Options{
		ProfileName: "work",
		Days:        7,
		Counts:      map[string]int{truncateToDay(time.Now()).Format("2006-01-02"): 5},
		NoColor:     true,
	})
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "\x1b[") {
		t.Fatalf("expected no ANSI escape codes with NoColor=true, got:\n%q", joined)
	}
}

func TestRender_ColorEnabledIncludesEscapeCodes(t *testing.T) {
	today := truncateToDay(time.Now()).Format("2006-01-02")
	lines := Render(Options{
		ProfileName: "work",
		Days:        7,
		Counts:      map[string]int{today: 5},
		NoColor:     false,
	})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "\x1b[") {
		t.Fatalf("expected ANSI escape codes with NoColor=false, got:\n%q", joined)
	}
}

func TestRender_DefaultsDaysWhenZero(t *testing.T) {
	lines := Render(Options{ProfileName: "work", Days: 0, Counts: map[string]int{}, NoColor: true})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "last 30 days") {
		t.Fatalf("expected Days<=0 to default to 30, got:\n%s", joined)
	}
}

func TestIntensityLevel(t *testing.T) {
	cases := []struct {
		count, max, want int
	}{
		{0, 10, 0},
		{1, 1, 4}, // sparse data: any activity maxes out
		{0, 0, 0},
		{1, 10, 1},
		{10, 10, 4},
		{5, 10, 2},
	}
	for _, c := range cases {
		if got := intensityLevel(c.count, c.max); got != c.want {
			t.Errorf("intensityLevel(%d, %d) = %d, want %d", c.count, c.max, got, c.want)
		}
	}
}

func TestVisibleWidth_StripsANSI(t *testing.T) {
	colored := levelColors[2] + "██" + colorReset
	if w := visibleWidth(colored); w != 2 {
		t.Fatalf("visibleWidth(%q) = %d, want 2", colored, w)
	}
}

func TestSideBySide_AlignsColumns(t *testing.T) {
	blockA := []string{"short", "line2"}
	blockB := []string{"a much longer first line", "b"}

	out := SideBySide([][]string{blockA, blockB}, " | ")
	if len(out) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(out))
	}
	// blockA's lines should be padded to the width of its longest line
	// ("short" == 5 chars), so both rows of the joined output should have
	// the separator at the same column.
	sepIdx0 := strings.Index(out[0], "|")
	sepIdx1 := strings.Index(out[1], "|")
	if sepIdx0 != sepIdx1 {
		t.Fatalf("expected aligned separators, got %d vs %d:\n%s\n%s", sepIdx0, sepIdx1, out[0], out[1])
	}
}

func TestSideBySide_PadsShorterBlocks(t *testing.T) {
	blockA := []string{"one", "two", "three"}
	blockB := []string{"only"}

	out := SideBySide([][]string{blockA, blockB}, "|")
	if len(out) != 3 {
		t.Fatalf("expected output to span the longest block's line count, got %d lines", len(out))
	}
}

func TestSideBySide_Empty(t *testing.T) {
	if out := SideBySide(nil, "|"); out != nil {
		t.Fatalf("expected nil for no blocks, got %v", out)
	}
}

func TestSideBySide_ColorAware(t *testing.T) {
	colored := []string{levelColors[4] + "██" + colorReset}
	plain := []string{"XX"}

	out := SideBySide([][]string{colored, plain}, "|")
	// The colored block's visible width is 2, matching "XX"; there should
	// be no extra padding spaces inserted before the separator.
	if !strings.Contains(out[0], "██"+colorReset+"|XX") {
		t.Fatalf("expected tight alignment accounting for ANSI width, got: %q", out[0])
	}
}
