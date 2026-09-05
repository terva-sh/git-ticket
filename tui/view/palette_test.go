package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
)

// These tests hold the row palettes to the TKT-01M1S022 decision:
// priority is the default, p cycles the set at runtime, and NO_COLOR
// pins the colors off, key included. They assert raw output on
// purpose: the color is the content here, so plain() would strip
// exactly what is under test.

func TestPaletteDefaultsToPriority(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "")
	idx, locked := startPalette()
	if palettes[idx].name != "priority" || locked {
		t.Fatalf("default = %q locked=%v, want priority unlocked", palettes[idx].name, locked)
	}
}

func TestPaletteUnknownNameIsTheDefault(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "solarized")
	idx, locked := startPalette()
	if palettes[idx].name != "priority" || locked {
		t.Fatalf("unknown name = %q locked=%v, want the priority default", palettes[idx].name, locked)
	}
}

func TestPaletteNoColorPinsOff(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "status")
	// Presence is the convention, even empty, per no-color.org.
	t.Setenv("NO_COLOR", "")
	idx, locked := startPalette()
	if palettes[idx].name != "off" || !locked {
		t.Fatalf("NO_COLOR gave %q locked=%v, want off locked", palettes[idx].name, locked)
	}

	// The p key refuses rather than cycling, and says why.
	v := newTestList(fixed(mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "high", "Hot work")))
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'p'})
	if got := palettes[v.paletteIdx].name; got != "off" {
		t.Fatalf("p cycled under NO_COLOR, to %q", got)
	}
	if foot := footerOf(v); !strings.Contains(foot, "NO_COLOR") {
		t.Fatalf("footer = %q, want the NO_COLOR explanation", foot)
	}
}

// TestPaletteKeyCyclesTheSet walks p through the whole ring and back,
// with the header naming each stop, because an invisible color mode
// reads as broken.
func TestPaletteKeyCyclesTheSet(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "")
	v := newTestList(fixed(mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "high", "Hot work")))

	want := []string{"status", "type", "dim", "off", "priority"}
	for _, name := range want {
		v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'p'})
		if got := palettes[v.paletteIdx].name; got != name {
			t.Fatalf("cycle reached %q, want %q", got, name)
		}
		head := plain(v, 110, 8)[0]
		if !strings.Contains(head, "color: "+name) {
			t.Fatalf("header = %q, want color: %s", head, name)
		}
	}
}

func TestPaletteColorsTheExceptionalRow(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "")
	// The normal ticket comes first so the cursor sits on it: the
	// selected row correctly suppresses the palette, which an earlier
	// version of this test learned by putting the colored row on top.
	ordinary := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "Ordinary work")
	hot := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "ready", "high", "Hot work")
	v := newTestList(fixed(ordinary, hot))
	rows := v.Render(110, 8)

	var hotRow, ordinaryRow string
	for _, r := range rows {
		if strings.Contains(r, "Hot work") {
			hotRow = r
		}
		if strings.Contains(r, "Ordinary work") {
			ordinaryRow = r
		}
	}
	if !strings.Contains(hotRow, sgrBlue) {
		t.Fatalf("the high row carries no blue under the default palette: %q", hotRow)
	}
	if strings.Contains(ordinaryRow, "38;5;") && !strings.Contains(ordinaryRow, "\x1b[7m") {
		t.Fatalf("the ordinary normal row is colored: %q", ordinaryRow)
	}
}

// TestPaletteSelectedRowStaysPlain is the ticket's legibility
// constraint: the cursor is reverse-video and nothing else, over every
// palette.
func TestPaletteSelectedRowStaysPlain(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "priority")
	hot := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "ready", "high", "Hot work")
	v := newTestList(fixed(hot))
	rows := v.Render(110, 8)

	var selected string
	for _, r := range rows {
		if strings.Contains(r, "\x1b[7m") {
			selected = r
		}
	}
	if selected == "" {
		t.Fatal("no selected row rendered")
	}
	if strings.Contains(selected, "38;5;") {
		t.Fatalf("the selected row carries palette color: %q", selected)
	}
}

// TestEveryPaletteLeavesTheOrdinaryAlone: a palette that colors
// everything makes color mean nothing, so a ready normal task renders
// plain under every palette.
func TestEveryPaletteLeavesTheOrdinaryAlone(t *testing.T) {
	ordinary := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "Ordinary")
	ordinary.Type = "task"
	for _, p := range palettes {
		if c := p.color(ordinary); c != "" {
			t.Errorf("palette %q colors the ordinary case with %q", p.name, c)
		}
	}
}
