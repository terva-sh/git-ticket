package tui_test

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
)

func TestWrapPlainBreaksAtSpaces(t *testing.T) {
	got := tui.WrapPlain("one two three four five", 9)
	want := []string{"one two", "three", "four five"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("WrapPlain = %v, want %v", got, want)
	}
}

func TestWrapPlainKeepsShortLines(t *testing.T) {
	got := tui.WrapPlain("short", 40)
	if len(got) != 1 || got[0] != "short" {
		t.Fatalf("WrapPlain = %v", got)
	}
	// Blank lines survive, because a paragraph break is content.
	got = tui.WrapPlain("a\n\nb", 40)
	if len(got) != 3 || got[1] != "" {
		t.Fatalf("WrapPlain lost the blank line: %v", got)
	}
}

func TestWrapPlainHardBreaksAWideWord(t *testing.T) {
	got := tui.WrapPlain("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("WrapPlain = %v, want %v", got, want)
	}
}

func TestWrapPlainKeepsIndentOnContinuations(t *testing.T) {
	got := tui.WrapPlain("  - a checklist item that runs long", 16)
	if len(got) < 2 {
		t.Fatalf("expected a wrap: %v", got)
	}
	for i, line := range got {
		if !strings.HasPrefix(line, "  ") {
			t.Fatalf("line %d lost the indent: %q", i, line)
		}
		if w := tui.VisibleWidth(line); w > 16 {
			t.Fatalf("line %d is %d cells wide: %q", i, w, line)
		}
	}
}

func TestWrapPlainCountsWideRunes(t *testing.T) {
	for _, line := range tui.WrapPlain("日本 日本 日本", 5) {
		if w := tui.VisibleWidth(line); w > 5 {
			t.Fatalf("line %q is %d cells wide, limit 5", line, w)
		}
	}
}
