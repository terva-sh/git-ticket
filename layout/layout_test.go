package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The point of this file is one property: moving a card changes one line.
// Everything else about the format follows from it, so it is the thing worth
// holding down with a test rather than a comment.
func TestOneDragIsOneLine(t *testing.T) {
	s := New(t.TempDir())
	b := &Board{Schema: Schema, Board: "default", Cards: map[string]Card{
		"TKT-A": {X: 0, Y: 0},
		"TKT-B": {X: 100, Y: 40},
		"TKT-C": {X: 200, Y: 80},
	}}
	if err := s.Save(b); err != nil {
		t.Fatal(err)
	}
	before := readLines(t, s, "default")

	if _, err := s.Update("default", map[string]*Card{"TKT-B": {X: 340, Y: 220}}); err != nil {
		t.Fatal(err)
	}
	after := readLines(t, s, "default")

	if len(before) != len(after) {
		t.Fatalf("line count changed: %d -> %d", len(before), len(after))
	}
	changed := 0
	for i := range before {
		if before[i] != after[i] {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("a single drag changed %d lines, want 1:\n%s\n---\n%s",
			changed, strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
}

// Order has to come from the IDs and not from Go's map iteration, or two
// writers with identical boards produce different bytes and every save is a
// whole-file diff.
func TestRenderIsDeterministicAndSorted(t *testing.T) {
	b := &Board{Schema: Schema, Board: "b", Cards: map[string]Card{
		"TKT-C": {X: 3}, "TKT-A": {X: 1}, "TKT-B": {X: 2},
	}}
	first := string(render(b))
	for i := 0; i < 20; i++ {
		if got := string(render(b)); got != first {
			t.Fatalf("render is not deterministic:\n%s\n---\n%s", first, got)
		}
	}
	ia := strings.Index(first, "TKT-A")
	ib := strings.Index(first, "TKT-B")
	ic := strings.Index(first, "TKT-C")
	if !(ia < ib && ib < ic) {
		t.Fatalf("cards are not sorted by ID:\n%s", first)
	}
}

// A board that does not exist is an empty board. The first card placed on a
// canvas must not have to be preceded by a command that creates the file.
func TestMissingBoardLoadsEmpty(t *testing.T) {
	s := New(t.TempDir())
	b, err := s.Load("never-written")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Cards) != 0 || b.Board != "never-written" || b.Schema != Schema {
		t.Fatalf("unexpected empty board: %+v", b)
	}
}

func TestRoundTrip(t *testing.T) {
	s := New(t.TempDir())
	in := &Board{Schema: Schema, Board: "default", Cards: map[string]Card{
		"TKT-A": {X: -12.5, Y: 0, W: 320, Z: 4, Collapsed: true},
		"TKT-B": {X: 1.005, Y: -0.004},
	}}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := s.Load("default")
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Cards["TKT-A"]; got != (Card{X: -12.5, W: 320, Z: 4, Collapsed: true}) {
		t.Fatalf("card A round-tripped as %+v", got)
	}
	// Sub-pixel noise is rounded away rather than written, so a position
	// nothing meaningfully moved does not rewrite its line.
	if got := out.Cards["TKT-B"]; got.X != 1 || got.Y != 0 {
		t.Fatalf("card B round-tripped as %+v, want x:1 y:0", got)
	}
}

// A card mapped to nil is removed. That is how a client says "off the board"
// without deleting a ticket, and it must not leave an entry behind.
func TestUpdateRemoves(t *testing.T) {
	s := New(t.TempDir())
	if _, err := s.Update("default", map[string]*Card{"TKT-A": {X: 1}, "TKT-B": {X: 2}}); err != nil {
		t.Fatal(err)
	}
	b, err := s.Update("default", map[string]*Card{"TKT-A": nil})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Cards["TKT-A"]; ok {
		t.Fatal("TKT-A survived removal")
	}
	if len(b.Cards) != 1 {
		t.Fatalf("want 1 card left, got %d", len(b.Cards))
	}
}

func TestRejectsBadBoardNames(t *testing.T) {
	s := New(t.TempDir())
	for _, name := range []string{"", "../escape", "a/b", "with space", strings.Repeat("x", 65)} {
		if _, err := s.Load(name); err == nil {
			t.Fatalf("board name %q was accepted", name)
		}
	}
}

func TestRefusesFutureSchema(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := os.MkdirAll(s.Dir(), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Dir(), "default.yml")
	if err := os.WriteFile(path, []byte("schema: 99\nboard: default\ncards: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load("default"); err == nil {
		t.Fatal("a board from a newer build was parsed rather than refused")
	}
}

func readLines(t *testing.T, s *Store, board string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(s.Dir(), board+".yml"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}
