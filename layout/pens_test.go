package layout

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Schema-3 contract tests intentionally use test-local wire data. They compile
// against schema 2 and fail on behavior, not on missing production Go symbols.
// Board fields are pens, ruleOrder, inbox; a pen has pin and requiredLabels.
func penSchemaFixture() map[string]any {
	return map[string]any{
		"schema": 3, "board": DefaultBoard,
		"cards":  map[string]any{memberA: map[string]any{"x": 30.25, "y": -40.5, "w": 310, "z": 2, "collapsed": true}},
		"frames": map[string]any{"release": map[string]any{"title": "Release", "x": 10, "y": 20, "w": 620, "h": 420, "color": "#759bcc", "members": []string{memberA, memberB}}},
		"pens": map[string]any{
			"bugs":   schemaPen("Frontend bugs", []string{"frontend", "bug"}),
			"urgent": schemaPen("Frontend urgent", []string{"frontend", "urgent"}),
		},
		"ruleOrder": []string{"urgent", "bugs"},
		"inbox":     map[string]any{"x": -400, "y": 20},
	}
}

func schemaPen(title string, labels []string) map[string]any {
	return map[string]any{"title": title, "x": 20, "y": 40, "w": 620, "h": 420,
		"color": "#759bcc", "pin": map[string]any{"x": 40, "y": 80}, "requiredLabels": labels}
}

func schemaYAML(t *testing.T, value any) []byte {
	t.Helper()
	data, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func schemaJSON(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func writeSchemaFixture(t *testing.T, s *Store, board string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(s.Dir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir(), board+".yml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPenSchema3RoundTripAndExistingWriters(t *testing.T) {
	fixture := penSchemaFixture()
	s := New(t.TempDir())
	original := schemaYAML(t, fixture)
	writeSchemaFixture(t, s, DefaultBoard, original)
	b, err := s.Load(DefaultBoard)
	if err != nil {
		t.Fatalf("valid schema-3 pen layout must load: %v", err)
	}
	want := schemaJSON(t, fixture)
	if got := schemaJSON(t, b); !reflect.DeepEqual(got, want) {
		t.Fatalf("wire round trip: got %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(bytesOnDisk(t, s), original) {
		t.Fatal("load rewrote authored data")
	}
	if err := s.Save(b); err != nil {
		t.Fatal(err)
	}
	canonical := bytesOnDisk(t, s)
	for range 3 {
		loaded, err := s.Load(DefaultBoard)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Save(loaded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(bytesOnDisk(t, s), canonical) {
			t.Fatal("canonical write is unstable")
		}
	}
	assertRouting := func(out *Board) {
		t.Helper()
		got := schemaJSON(t, out)
		for _, field := range []string{"pens", "ruleOrder", "inbox"} {
			if !reflect.DeepEqual(got[field], want[field]) {
				t.Fatalf("writer lost %s: %#v", field, got[field])
			}
		}
	}
	out, err := s.Update(DefaultBoard, map[string]*Card{memberB: {X: 100, Y: 200}})
	if err != nil {
		t.Fatal(err)
	}
	assertRouting(out)
	f := out.Frames["release"]
	renamed := f
	renamed.Title = "Next release"
	beforeCard := b.Cards[memberA]
	out, err = s.Transaction(DefaultBoard, map[string]*Card{memberA: nil}, map[string]*Frame{"release": &renamed},
		&Expectations{Cards: map[string]*Card{memberA: &beforeCard}, Frames: map[string]*Frame{"release": &f}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertRouting(out)
	if _, exists := out.Cards[memberA]; exists {
		t.Fatal("unpin retained manual position")
	}
	if !reflect.DeepEqual(out.Frames["release"].Members, f.Members) {
		t.Fatal("unpin changed membership")
	}
	out, err = s.RemoveTicket(DefaultBoard, memberB)
	if err != nil {
		t.Fatal(err)
	}
	assertRouting(out)
	if !reflect.DeepEqual(out.Frames["release"].Members, []string{memberA}) {
		t.Fatal("deletion did not clean only its membership")
	}
}

func TestPenSchemaLegacyReadsDoNotWriteAndMutationUpgrades(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			s := New(t.TempDir())
			fixture := penSchemaFixture()
			fixture["schema"] = version
			delete(fixture, "pens")
			delete(fixture, "ruleOrder")
			delete(fixture, "inbox")
			if version == 1 {
				delete(fixture, "frames")
			}
			original := schemaYAML(t, fixture)
			writeSchemaFixture(t, s, DefaultBoard, original)
			b, err := s.Load(DefaultBoard)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(bytesOnDisk(t, s), original) {
				t.Fatal("legacy read changed disk")
			}
			wire := schemaJSON(t, b)
			if b.Schema != 3 {
				t.Fatalf("normalized schema=%d, want 3", b.Schema)
			}
			if !reflect.DeepEqual(wire["pens"], map[string]any{}) || !reflect.DeepEqual(wire["ruleOrder"], []any{}) {
				t.Fatal("legacy board must expose empty routing records")
			}
			if _, ok := wire["inbox"].(map[string]any); !ok {
				t.Fatal("legacy board lacks default Inbox")
			}
			beforeCards, beforeFrames := b.Cards, b.Frames
			out, err := s.Update(DefaultBoard, map[string]*Card{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(out.Cards, beforeCards) || !reflect.DeepEqual(out.Frames, beforeFrames) {
				t.Fatal("migration changed manual cards or frames")
			}
			var disk map[string]any
			if err := yaml.Unmarshal(bytesOnDisk(t, s), &disk); err != nil {
				t.Fatal(err)
			}
			if disk["schema"] != 3 {
				t.Fatalf("mutation did not persist schema 3: %v", disk["schema"])
			}
		})
	}
}

func TestPenSchemaLabelsPreserveIdentityAndDeduplicate(t *testing.T) {
	fixture := penSchemaFixture()
	labels := []string{"Frontend", "frontend", "frontend", "two words", "comma,label", "unused-label", "null"}
	fixture["pens"].(map[string]any)["bugs"].(map[string]any)["requiredLabels"] = labels
	b, err := Parse(DefaultBoard, schemaYAML(t, fixture))
	if err != nil {
		t.Fatalf("nonempty exact labels must be accepted: %v", err)
	}
	got := schemaJSON(t, b)["pens"].(map[string]any)["bugs"].(map[string]any)["requiredLabels"].([]any)
	counts := map[string]int{}
	for _, label := range got {
		counts[label.(string)]++
	}
	want := map[string]int{"Frontend": 1, "frontend": 1, "two words": 1, "comma,label": 1, "unused-label": 1, "null": 1}
	if !reflect.DeepEqual(counts, want) {
		t.Fatalf("labels were split, rewritten, lost, or weighted twice: %v", counts)
	}
}

func TestPenSchemaSensitiveKeysAndStrings(t *testing.T) {
	for _, id := range []string{"null", "NULL", "true", "123", "-"} {
		t.Run(id, func(t *testing.T) {
			fixture := penSchemaFixture()
			fixture["board"] = id
			fixture["pens"] = map[string]any{id: schemaPen("Next: \"release\" #1", []string{"null", "a: b", "x,y"})}
			fixture["ruleOrder"] = []string{id}
			b, err := Parse(id, schemaYAML(t, fixture))
			if err != nil {
				t.Fatal(err)
			}
			s := New(t.TempDir())
			if err := s.Save(b); err != nil {
				t.Fatal(err)
			}
			loaded, err := s.Load(id)
			if err != nil || !reflect.DeepEqual(schemaJSON(t, loaded), schemaJSON(t, b)) {
				t.Fatalf("sensitive key/string lost: %v", err)
			}
		})
	}
}

func TestPenSchemaInvalidLayoutsCannotBeRewritten(t *testing.T) {
	// A positive control prevents schema-2's blanket rejection from looking like
	// successful validation of malformed schema-3 pen records.
	if _, err := Parse(DefaultBoard, schemaYAML(t, penSchemaFixture())); err != nil {
		t.Fatalf("positive control: %v", err)
	}
	cases := map[string]func(map[string]any){
		"empty rule": func(b map[string]any) {
			b["pens"].(map[string]any)["bugs"].(map[string]any)["requiredLabels"] = []string{}
		},
		"null rule": func(b map[string]any) { b["pens"].(map[string]any)["bugs"].(map[string]any)["requiredLabels"] = nil },
		"blank label": func(b map[string]any) {
			b["pens"].(map[string]any)["bugs"].(map[string]any)["requiredLabels"] = []string{" "}
		},
		"missing order member":   func(b map[string]any) { b["ruleOrder"] = []string{"bugs"} },
		"duplicate order member": func(b map[string]any) { b["ruleOrder"] = []string{"bugs", "bugs", "urgent"} },
		"unknown order member":   func(b map[string]any) { b["ruleOrder"] = []string{"bugs", "urgent", "missing"} },
		"missing pens":           func(b map[string]any) { delete(b, "pens") },
		"null pens":              func(b map[string]any) { b["pens"] = nil },
		"null order":             func(b map[string]any) { b["ruleOrder"] = nil },
		"partial Inbox":          func(b map[string]any) { b["inbox"] = map[string]any{"x": 0} },
		"unknown Inbox field":    func(b map[string]any) { b["inbox"].(map[string]any)["future"] = true },
		"missing pin":            func(b map[string]any) { delete(b["pens"].(map[string]any)["bugs"].(map[string]any), "pin") },
		"null pin":               func(b map[string]any) { b["pens"].(map[string]any)["bugs"].(map[string]any)["pin"] = nil },
		"partial pin": func(b map[string]any) {
			b["pens"].(map[string]any)["bugs"].(map[string]any)["pin"] = map[string]any{"x": 0}
		},
		"nonfinite pin": func(b map[string]any) {
			b["pens"].(map[string]any)["bugs"].(map[string]any)["pin"] = map[string]any{"x": math.Inf(1), "y": 0}
		},
		"null coordinate": func(b map[string]any) { b["pens"].(map[string]any)["bugs"].(map[string]any)["x"] = nil },
		"missing Inbox":   func(b map[string]any) { delete(b, "inbox") },
		"null Inbox":      func(b map[string]any) { b["inbox"] = nil },
		"bad geometry":    func(b map[string]any) { b["pens"].(map[string]any)["bugs"].(map[string]any)["w"] = 0 },
		"unknown field":   func(b map[string]any) { b["pens"].(map[string]any)["bugs"].(map[string]any)["future"] = true },
		"future schema":   func(b map[string]any) { b["schema"] = 4 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			fixture := penSchemaFixture()
			change(fixture)
			data := schemaYAML(t, fixture)
			s := New(t.TempDir())
			writeSchemaFixture(t, s, DefaultBoard, data)
			if _, err := s.Load(DefaultBoard); err == nil {
				t.Fatal("accepted malformed layout")
			}
			if _, err := s.Update(DefaultBoard, nil); err == nil {
				t.Fatal("rewrote malformed layout")
			}
			if err := s.Save(Empty(DefaultBoard)); err == nil {
				t.Fatal("overwrote malformed layout")
			}
			if !reflect.DeepEqual(bytesOnDisk(t, s), data) {
				t.Fatal("invalid layout changed disk")
			}
		})
	}
	// Empty pens are valid. An empty rule inside a pen is not.
	fixture := penSchemaFixture()
	fixture["pens"] = map[string]any{}
	fixture["ruleOrder"] = []string{}
	if _, err := Parse(DefaultBoard, schemaYAML(t, fixture)); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(schemaYAML(t, fixture)), "schema: 3", "schema: 2", 1)
	if _, err := Parse(DefaultBoard, []byte(bad)); err == nil {
		t.Fatal("schema 2 must not smuggle schema-3 routing fields")
	}
}
