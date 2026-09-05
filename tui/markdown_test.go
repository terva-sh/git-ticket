package tui_test

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
)

// TestRenderMarkdownHeadingIsBoldAndAccented pins the heading style:
// bold plus the theme accent, with the prose intact underneath.
func TestRenderMarkdownHeadingIsBoldAndAccented(t *testing.T) {
	out := tui.RenderMarkdown("### Rollback plan", tui.DefaultTheme, 80)
	if !strings.Contains(out, "\x1b[1m") {
		t.Errorf("heading is not bold: %q", out)
	}
	if !strings.Contains(out, "\x1b[38;5;111m") {
		t.Errorf("heading does not carry the accent color: %q", out)
	}
	if got := tui.StripANSI(out); got != "### Rollback plan" {
		t.Errorf("StripANSI lost the heading prose: %q", got)
	}
}

// TestRenderMarkdownStylesTheBlocks walks the block kinds the detail
// view meets in real tickets: bullets, blockquotes, and inline code.
func TestRenderMarkdownStylesTheBlocks(t *testing.T) {
	src := "- first point\n> a quote\nrun `just ci` before pushing"
	out := tui.RenderMarkdown(src, tui.DefaultTheme, 80)
	plain := tui.StripANSI(out)
	for _, want := range []string{"• first point", "┃ a quote", "run just ci before pushing"} {
		if !strings.Contains(plain, want) {
			t.Errorf("styled render lost %q:\n%s", want, plain)
		}
	}
	if !strings.Contains(out, "\x1b[38;5;111mjust ci\x1b[0m") {
		t.Errorf("inline code is not accented: %q", out)
	}
}

// TestRenderMarkdownHighlightsAGoFence is the chroma criterion: a
// fenced block with a language hint comes back colored, and the code
// itself survives byte for byte under StripANSI.
func TestRenderMarkdownHighlightsAGoFence(t *testing.T) {
	code := "func main() {\n\treturn\n}"
	out := tui.RenderMarkdown("```go\n"+code+"\n```", tui.DefaultTheme, 80)
	if got := tui.StripANSI(out); got != code {
		t.Errorf("StripANSI of the highlighted fence is not the code:\ngot  %q\nwant %q", got, code)
	}
	// chroma's terminal256 formatter colors the func keyword; any SGR
	// sequence in the output proves the fence went through it, because
	// an unhighlighted fallback is byte-identical to the input.
	if !strings.Contains(out, "\x1b[38;5;") {
		t.Errorf("go fence came back with no 256-color styling: %q", out)
	}
}

// TestRenderMarkdownFenceWithoutLangIsAccented pins the fallback: no
// language hint means no chroma, but the block still reads as code.
func TestRenderMarkdownFenceWithoutLangIsAccented(t *testing.T) {
	out := tui.RenderMarkdown("```\nplain block\n```", tui.DefaultTheme, 80)
	if !strings.Contains(out, "\x1b[38;5;111mplain block\x1b[0m") {
		t.Errorf("bare fence is not accent-colored: %q", out)
	}
}

// TestRenderMarkdownTableAlignsCells pins the table path: cells pad to
// a shared width and the pipes survive, so columns line up.
func TestRenderMarkdownTableAlignsCells(t *testing.T) {
	src := "| Phase | State |\n| --- | --- |\n| 0 | done |\n| 4 | started |"
	out := tui.RenderMarkdown(src, tui.DefaultTheme, 80)
	lines := strings.Split(tui.StripANSI(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("table rendered %d lines, want 4:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	for i, l := range lines {
		if !strings.HasPrefix(l, "|") || !strings.HasSuffix(l, "|") {
			t.Errorf("row %d lost its pipes: %q", i, l)
		}
		if w := tui.VisibleWidth(l); w != tui.VisibleWidth(lines[0]) {
			t.Errorf("row %d is %d wide, header is %d: columns do not align", i, w, tui.VisibleWidth(lines[0]))
		}
	}
}
