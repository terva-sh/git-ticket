package layout

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const memberA = "TKT-01M24411DDC98WXKT2MY2FMHQN"
const memberB = "TKT-01M24411DDC98WXKT2MY2FMHQP"

func testFrame(members ...string) Frame {
	return Frame{Title: "Delivery: \"next\"", X: -10, Y: 20, W: 620, H: 420, Color: "#759bcc", Members: append([]string{}, members...)}
}
func frameBoard(t *testing.T) (*Store, *Board) {
	t.Helper()
	s := New(t.TempDir())
	b := Empty(DefaultBoard)
	b.Cards[memberA] = Card{X: 30, Y: 40}
	b.Frames["f"] = testFrame(memberA, memberB)
	if err := s.Save(b); err != nil {
		t.Fatal(err)
	}
	return s, b
}
func bytesOnDisk(t *testing.T, s *Store) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(s.Dir(), "default.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestFrameRoundTripDeterministic(t *testing.T) {
	s, b := frameBoard(t)
	f := testFrame(memberB, memberA)
	b.Frames["a"] = testFrame()
	b.Frames["f"] = f
	if err := s.Save(b); err != nil {
		t.Fatal(err)
	}
	before := bytesOnDisk(t, s)
	for i := 0; i < 20; i++ {
		if string(render(b)) != string(before) {
			t.Fatal("unstable rendering")
		}
	}
	if a, f := strings.Index(string(before), `  "a":`), strings.Index(string(before), `  "f":`); a < 0 || f < 0 || a >= f {
		t.Fatal("unsorted frames")
	}
	loaded, err := s.Load(DefaultBoard)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.Frames["f"].Members, []string{memberA, memberB}) {
		t.Fatal("unsorted members")
	}
	if loaded.Frames["f"].Title != f.Title || loaded.Cards[memberA] != b.Cards[memberA] {
		t.Fatal("round trip lost data")
	}
	if _, err := s.Update(DefaultBoard, map[string]*Card{memberB: {X: 88}}); err != nil {
		t.Fatal(err)
	}
	loaded, _ = s.Load(DefaultBoard)
	if !sameFrame(loaded.Frames["f"], f) {
		t.Fatal("sparse card update lost frame")
	}
	loaded, err = s.Update(DefaultBoard, map[string]*Card{memberB: nil})
	if err != nil {
		t.Fatal(err)
	}
	if !sameFrame(loaded.Frames["f"], f) {
		t.Fatal("card-only removal lost membership")
	}
}

func TestYAMLSensitiveIdentifiersRoundTrip(t *testing.T) {
	for _, name := range []string{"null", "Null", "NULL", "true", "false", "yes", "on", "123", "0x10", "2026-09-10", "-", "_"} {
		t.Run(name, func(t *testing.T) {
			s := New(t.TempDir())
			frame := testFrame(memberA)
			card := Card{X: 12.5, Y: -7.25}
			// Board names and both map-key types accept these strings. None
			// may become a YAML null, boolean, number, or timestamp on disk.
			written, err := s.Transaction(name, map[string]*Card{name: &card}, map[string]*Frame{name: &frame},
				&Expectations{Cards: map[string]*Card{name: nil}, Frames: map[string]*Frame{name: nil}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.Dir(), name+".yml")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				loaded, err := s.Load(name)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(loaded, written) {
					t.Fatalf("round trip changed board: got %#v, want %#v", loaded, written)
				}
				if _, err := s.Update(name, nil); err != nil {
					t.Fatal(err)
				}
				after, err := os.ReadFile(path)
				if err != nil || string(before) != string(after) {
					t.Fatalf("repeat write changed canonical bytes: %v", err)
				}
			}
		})
	}
	// Isolate map keys from board-name validation so a dropped frame or
	// card cannot hide behind a board parse error.
	s := New(t.TempDir())
	b := Empty(DefaultBoard)
	b.Frames["null"] = testFrame(memberA)
	b.Cards["NULL"] = Card{X: 20, Y: 40}
	if err := s.Save(b); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(DefaultBoard)
	if err != nil || !reflect.DeepEqual(loaded, b) {
		t.Fatalf("null map keys lost data: got %#v, error %v", loaded, err)
	}
}

func TestLayoutSchemaCompatibilityAndFailSafe(t *testing.T) {
	good := "schema: 1\nboard: default\ncards:\n  TKT-A: {x: 12.5123, y: -3}\n"
	b, err := Parse(DefaultBoard, []byte(good))
	if err != nil || b.Frames == nil || b.Schema != Schema || b.Cards["TKT-A"].X != 12.5123 {
		t.Fatalf("old layout: %+v %v", b, err)
	}
	for _, data := range []string{
		strings.Replace(good, "schema: 1", fmt.Sprintf("schema: %d", Schema+1), 1),
		good + "future: {}\n",
		strings.Replace(good, "y: -3", "y: -3, future: true", 1),
		good + "---\nboard: other\n",
		strings.Replace(good, "board: default", "board: other", 1),
		strings.Replace(good, "schema: 1", "schema: 0", 1),
		string(render(&Board{Schema: Schema, Board: DefaultBoard, Frames: map[string]Frame{"f": testFrame()}})) + "extra: 2\n",
		strings.Replace(string(render(&Board{Schema: Schema, Board: DefaultBoard, Frames: map[string]Frame{"f": testFrame()}})), "title:", "future: true, title:", 1),
	} {
		t.Run(fmt.Sprint(len(data), data[:9]), func(t *testing.T) {
			s := New(t.TempDir())
			if err := os.MkdirAll(s.Dir(), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(s.Dir(), "default.yml"), []byte(data), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Load(DefaultBoard); err == nil {
				t.Fatal("accepted unsupported data")
			}
			if _, err := s.Update(DefaultBoard, nil); err == nil {
				t.Fatal("updated unsupported data")
			}
			if err := s.Save(Empty(DefaultBoard)); err == nil {
				t.Fatal("overwrote unsupported data")
			}
			if string(bytesOnDisk(t, s)) != data {
				t.Fatal("failed write changed bytes")
			}
		})
	}
}

func TestFrameConflictIsAtomicAndChecksExtraReadSet(t *testing.T) {
	s, b := frameBoard(t)
	f := b.Frames["f"]
	moved := f
	moved.X += 100
	before := bytesOnDisk(t, s)
	for name, expect := range map[string]*Expectations{
		"missing expect":  nil,
		"uncovered card":  {Frames: map[string]*Frame{"f": &f}},
		"uncovered frame": {Cards: map[string]*Card{memberA: {X: 30, Y: 40}}},
		"stale card":      {Cards: map[string]*Card{memberA: {X: 31}}, Frames: map[string]*Frame{"f": &f}},
		"expected null":   {Cards: map[string]*Card{memberA: nil}, Frames: map[string]*Frame{"f": &f}},
		"extra read":      {Cards: map[string]*Card{memberA: {X: 30, Y: 40}}, Frames: map[string]*Frame{"f": &f, "missing": &f}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.Transaction(DefaultBoard, map[string]*Card{memberA: {X: 130, Y: 40}}, map[string]*Frame{"f": &moved}, expect, nil)
			if err == nil {
				t.Fatal("accepted invalid transaction")
			}
			if strings.HasPrefix(name, "stale") || name == "expected null" || name == "extra read" {
				if !errors.Is(err, ErrConflict) {
					t.Fatalf("not conflict: %v", err)
				}
			}
			if string(bytesOnDisk(t, s)) != string(before) {
				t.Fatal("partial write")
			}
		})
	}
	// New records must compare absent and can be created in the same transaction.
	created := testFrame()
	if _, err := s.Transaction(DefaultBoard, map[string]*Card{memberB: {X: 4}}, map[string]*Frame{"new": &created}, &Expectations{Cards: map[string]*Card{memberB: nil}, Frames: map[string]*Frame{"new": nil}}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFrameSimultaneousEditsHaveOneWinner(t *testing.T) {
	s, b := frameBoard(t)
	f := b.Frames["f"]
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, dx := range []float64{100, 200} {
		wg.Add(1)
		go func(dx float64) {
			defer wg.Done()
			next := f
			next.X += dx
			_, err := s.Transaction(DefaultBoard, map[string]*Card{memberA: {X: 30 + dx, Y: 40}}, map[string]*Frame{"f": &next}, &Expectations{Cards: map[string]*Card{memberA: {X: 30, Y: 40}}, Frames: map[string]*Frame{"f": &f}}, nil)
			results <- err
		}(dx)
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	out, _ := s.Load(DefaultBoard)
	if out.Cards[memberA].X-30 != out.Frames["f"].X-f.X {
		t.Fatal("partial group movement")
	}
}

func TestFrameMembershipTransferAndManualUndo(t *testing.T) {
	s, b := frameBoard(t)
	f := b.Frames["f"]
	// Membership is independent of manual placement, including nil-card undo.
	if _, err := s.Transaction(DefaultBoard, map[string]*Card{memberA: nil}, nil, &Expectations{Cards: map[string]*Card{memberA: {X: 30, Y: 40}}, Frames: map[string]*Frame{"f": &f}}, nil); err != nil {
		t.Fatal(err)
	}
	out, _ := s.Load(DefaultBoard)
	if !sameFrame(out.Frames["f"], f) {
		t.Fatal("card removal removed membership")
	}
	target := testFrame(memberA)
	from := f
	from.Members = []string{memberB}
	if _, err := s.Transaction(DefaultBoard, nil, map[string]*Frame{"f": &from, "target": &target}, &Expectations{Frames: map[string]*Frame{"f": &f, "target": nil}}, nil); err != nil {
		t.Fatal(err)
	}
	// An old move must not move the former member after its transfer.
	_, err := s.Transaction(DefaultBoard, map[string]*Card{memberA: {X: 99}}, map[string]*Frame{"f": &f}, &Expectations{Cards: map[string]*Card{memberA: nil}, Frames: map[string]*Frame{"f": &f}}, nil)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale membership accepted: %v", err)
	}
	out, err = s.RemoveTicket(DefaultBoard, memberA)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Frames["target"].Members) != 0 || len(out.Frames["f"].Members) != 1 {
		t.Fatal("delete cleanup failed")
	}
}

func TestFrameValidationLeavesDiskUntouched(t *testing.T) {
	s, b := frameBoard(t)
	before := bytesOnDisk(t, s)
	f := b.Frames["f"]
	for name, change := range map[string]func(*Frame){
		"blank title":      func(f *Frame) { f.Title = " " },
		"long title":       func(f *Frame) { f.Title = strings.Repeat("a", 81) },
		"control title":    func(f *Frame) { f.Title = "a\nb" },
		"color":            func(f *Frame) { f.Color = "red" },
		"zero width":       func(f *Frame) { f.W = 0 },
		"negative height":  func(f *Frame) { f.H = -1 },
		"infinity":         func(f *Frame) { f.X = math.Inf(1) },
		"nan":              func(f *Frame) { f.Y = math.NaN() },
		"invalid member":   func(f *Frame) { f.Members = []string{"TKT-not-real"} },
		"duplicate member": func(f *Frame) { f.Members = []string{memberA, memberA} },
	} {
		t.Run(name, func(t *testing.T) {
			next := f
			change(&next)
			if _, err := s.Transaction(DefaultBoard, nil, map[string]*Frame{"f": &next}, &Expectations{Frames: map[string]*Frame{"f": &f}}, nil); err == nil {
				t.Fatal("accepted invalid frame")
			}
			if string(bytesOnDisk(t, s)) != string(before) {
				t.Fatal("invalid write changed disk")
			}
		})
	}
	duplicate := testFrame(memberA)
	if _, err := s.Transaction(DefaultBoard, nil, map[string]*Frame{"second": &duplicate}, &Expectations{Frames: map[string]*Frame{"second": nil}}, nil); err == nil {
		t.Fatal("duplicate assignment accepted")
	}
}
