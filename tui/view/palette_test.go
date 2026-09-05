package view

import (
	"strings"
	"testing"
)

// These tests hold the TKT-01M1S022 session scaffolding to the
// ticket's constraints. They assert raw output on purpose: the color
// is the content here, so plain() would strip exactly what is under
// test.

func TestPaletteDefaultsOff(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "")
	v := newTestList(fixed(mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "in-progress", "urgent", "Hot work")))
	rows := v.Render(100, 8)
	for _, r := range rows[1:] {
		if strings.Contains(r, "38;5;") {
			t.Fatalf("a row carries color with no palette asked for: %q", r)
		}
	}
}

func TestPaletteUnknownNameIsOff(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "solarized")
	if got := activePalette().name; got != "off" {
		t.Fatalf("unknown palette resolved to %q, want off", got)
	}
}

func TestPaletteNoColorWins(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "status")
	t.Setenv("NO_COLOR", "")
	// Presence is the convention, even empty, per no-color.org.
	if got := activePalette().name; got != "off" {
		t.Fatalf("NO_COLOR did not win: got %q", got)
	}
}

func TestPaletteColorsTheExceptionalRow(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "status")
	// The ready ticket comes first so the cursor sits on it: the
	// selected row correctly suppresses the palette, which the first
	// version of this test learned by putting blocked at the top.
	ready := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "Ordinary work")
	blocked := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "blocked", "normal", "Waiting on vendor")
	v := newTestList(fixed(ready, blocked))
	rows := v.Render(100, 8)

	var blockedRow, readyRow string
	for _, r := range rows {
		if strings.Contains(r, "Waiting on vendor") {
			blockedRow = r
		}
		if strings.Contains(r, "Ordinary work") {
			readyRow = r
		}
	}
	if !strings.Contains(blockedRow, sgrYellow) {
		t.Fatalf("the blocked row carries no yellow: %q", blockedRow)
	}
	if strings.Contains(readyRow, "38;5;") && !strings.Contains(readyRow, "\x1b[7m") {
		t.Fatalf("the ordinary ready row is colored: %q", readyRow)
	}
}

// TestPaletteSelectedRowStaysPlain is the ticket's legibility
// constraint: the cursor is reverse-video and nothing else, over every
// candidate.
func TestPaletteSelectedRowStaysPlain(t *testing.T) {
	t.Setenv("GIT_TICKET_UI_PALETTE", "status")
	blocked := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "blocked", "normal", "Waiting on vendor")
	v := newTestList(fixed(blocked))
	rows := v.Render(100, 8)

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

// TestEveryPaletteLeavesTheOrdinaryAlone: a candidate that colors
// everything makes color mean nothing, so a ready normal task renders
// plain under every candidate.
func TestEveryPaletteLeavesTheOrdinaryAlone(t *testing.T) {
	ordinary := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "Ordinary")
	ordinary.Type = "task"
	for _, p := range palettes {
		if c := p.color(ordinary); c != "" {
			t.Errorf("palette %q colors the ordinary case with %q", p.name, c)
		}
	}
}
