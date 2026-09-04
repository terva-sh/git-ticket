package tui_test

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
)

func TestVisibleWidthIgnoresEscapes(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"plain", 5},
		{"\x1b[1mbold\x1b[0m", 4},
		{"\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\", 4},
		{"日本", 4}, // wide runes are two cells each
	}
	for _, c := range cases {
		if got := tui.VisibleWidth(c.in); got != c.want {
			t.Errorf("VisibleWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestStripANSIRemovesCSIAndOSC(t *testing.T) {
	in := "\x1b[31mred\x1b[0m and \x1b]8;;https://example.com\x07link\x1b]8;;\x07"
	if got := tui.StripANSI(in); got != "red and link" {
		t.Fatalf("StripANSI = %q", got)
	}
}

func TestTruncateToWidthKeepsEscapesAndClipsCells(t *testing.T) {
	in := "\x1b[1m0123456789\x1b[0m"
	got := tui.TruncateToWidth(in, 4)
	if want := "\x1b[1m0123"; !strings.HasPrefix(got, want) {
		t.Fatalf("TruncateToWidth = %q, want prefix %q", got, want)
	}
	if tui.VisibleWidth(got) != 4 {
		t.Fatalf("clipped width = %d, want 4", tui.VisibleWidth(got))
	}
}

func TestTruncateToWidthClosesAnOpenHyperlink(t *testing.T) {
	in := "\x1b]8;;https://example.com\x1b\\a long linked run of text\x1b]8;;\x1b\\"
	got := tui.TruncateToWidth(in, 6)
	if !strings.HasSuffix(got, "\x1b]8;;\x1b\\") {
		t.Fatalf("clip left an OSC 8 link open: %q", got)
	}
	if tui.VisibleWidth(got) != 6 {
		t.Fatalf("clipped width = %d, want 6", tui.VisibleWidth(got))
	}
}

func TestTruncateToWidthDoesNotSplitAWideRune(t *testing.T) {
	got := tui.TruncateToWidth("日本語", 3)
	// Two cells for the first rune, and the second would need two more.
	if got != "日" {
		t.Fatalf("TruncateToWidth = %q, want %q", got, "日")
	}
}
