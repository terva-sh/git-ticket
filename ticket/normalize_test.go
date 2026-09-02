package ticket

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestNormalizeIsTheFixedPointOfParse is the claim the whole rule rests on: a
// body that came out of parse is already normal, so normalizing on the way to
// disk can never change a ticket that was read from one.
//
// It runs over the round-trip corpus rather than a handful of literals, because
// a normalizer that trims one field parse does not, or misses one parse does,
// is wrong in a way a hand-written example is unlikely to catch.
func TestNormalizeIsTheFixedPointOfParse(t *testing.T) {
	for _, path := range fixtures(t, filepath.Join(corpusDir, "parse", "roundtrip")) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			tk, err := Parse(data)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			before := tk.Body
			tk.Body.normalize()
			if !reflect.DeepEqual(before, tk.Body) {
				t.Errorf("normalize changed a parsed body\nbefore: %#v\nafter:  %#v", before, tk.Body)
			}
		})
	}
}

// TestWriteNormalizesEverySectionParseTrims is the guarantee moving to one
// place. A Ticket built in Go can be padded in any section, and every write
// funnels through writeTicket, so that is where it is settled.
//
// The second assertion is the one that decided where the normalization belongs.
// writeTicket returns the same Ticket it rendered, callers read that struct and
// the CLI serializes it, so the struct and the file have to agree. Normalizing
// inside Render would have left them disagreeing, which is a worse bug than the
// one it fixed because nothing shows it.
func TestWriteNormalizesEverySectionParseTrims(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Padded in every section")

	tk.Body = Body{
		Preamble:           "\n\nA preamble.\n\n",
		Description:        "\n\nThe description.\n\n",
		AcceptanceCriteria: "\n- [ ] The verifier accepts either key\n\n",
		DefinitionOfDone:   "\n\n- [ ] The runbook is updated\n",
		ImplementationPlan: "\n\n1. Read the code.\n\n",
		Notes:              "\n\nThe skew is 40s.\n\n",
		Comments:           "\n\nSecond pair of eyes wanted.\n\n",
		Summary:            "\n\nLanded in v1.2.\n\n",
		Extra:              []Section{{Heading: "  Risks  ", Text: "\n\nThe rollout is the risk.\n\n"}},
	}

	res, err := s.writeTicket(tk, tk.Path)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatal(err)
	}
	fromDisk, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// The bytes are the bytes a second render would produce, which is the
	// round trip plan 5.3 requires.
	if got := string(Render(fromDisk)); got != string(data) {
		t.Errorf("a padded body does not round trip\n%s", diffLines(string(data), got))
	}
	// The struct handed back describes the file that was written.
	if !reflect.DeepEqual(res.Ticket.Body, fromDisk.Body) {
		t.Errorf("the returned body disagrees with the file\nstruct: %#v\nfile:   %#v",
			res.Ticket.Body, fromDisk.Body)
	}
	if res.Ticket.Body.Description != "The description." {
		t.Errorf("description = %q", res.Ticket.Body.Description)
	}
	if len(res.Ticket.Body.Extra) != 1 || res.Ticket.Body.Extra[0].Heading != "Risks" {
		t.Errorf("extra = %#v, want the heading trimmed too", res.Ticket.Body.Extra)
	}
}

// TestWriteKeepsIndentationInsideASection is why the normalizer is
// trimBlankLines and not TrimSpace.
//
// Only blank lines at the edges of a section break the round trip. Leading
// whitespace on a content line survives a parse untouched, so trimming it would
// fix nothing and would silently reindent a section that opens with an indented
// code block. The per-writer TrimSpace this replaced did exactly that.
func TestWriteKeepsIndentationInsideASection(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Opens with an indented code block")

	const indented = "    git ticket ready\n    git ticket show ID\n\nThat is the queue."
	res, err := s.Apply(context.Background(), tk.ID, SetDescription{Text: indented}, ApplyOptions{Actor: testActor})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Ticket.Body.Description != indented {
		t.Errorf("the indentation was stripped:\n%q", res.Ticket.Body.Description)
	}

	data, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "    git ticket ready") {
		t.Errorf("the file lost the indentation:\n%s", data)
	}
	fromDisk, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := string(Render(fromDisk)); got != string(data) {
		t.Errorf("an indented section does not round trip\n%s", diffLines(string(data), got))
	}
}

// TestBatchedChecklistEditsLeaveNoInteriorPadding covers the one trim that did
// not move. normalize only reaches the edges of a section, so a removal that
// left blank lines behind and an add that appended after them would strand the
// padding in the middle, where nothing trims it. removeChecklistItem trims its
// own result for that reason.
func TestBatchedChecklistEditsLeaveNoInteriorPadding(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Prose above the list")

	// Seeded through the store, because no mutation sets a checklist section
	// wholesale and the case needs prose sitting above the list.
	tk.Body.AcceptanceCriteria = "The rollout has to hold these:\n\n- [ ] The verifier accepts either key"
	if _, err := s.writeTicket(tk, tk.Path); err != nil {
		t.Fatalf("seed: %v", err)
	}

	res := mustApply(t, s, tk.ID, Mutations{
		RemoveChecklistItem{Section: AcceptanceCriteria, Index: 1},
		AddChecklistItem{Section: AcceptanceCriteria, Text: "New tokens use the newer key"},
	})

	got := res.Ticket.Body.AcceptanceCriteria
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("a remove then an add left interior padding:\n%q", got)
	}
	data, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatal(err)
	}
	fromDisk, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out := string(Render(fromDisk)); out != string(data) {
		t.Errorf("a batched checklist edit does not round trip\n%s", diffLines(string(data), out))
	}
}
