package layout

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Modify loads a board, hands it to fn, and writes it back if what fn left is
// a board this package would accept. It is how `git ticket canvas` writes,
// per plan 12.10: one read-modify-write under the package mutex and the
// canvas directory's file lock, validated before the rename, so a refused
// write leaves the file exactly as it was and a concurrent writer waits
// rather than overwrites.
//
// fn edits the board in place and returns an error to refuse. It gets a board
// that Load already normalised, so a card it did not touch renders as it did.
func (s *Store) Modify(board string, fn func(*Board) error) (*Board, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	b, err := s.Load(board)
	if err != nil {
		return nil, err
	}
	if err := fn(b); err != nil {
		return nil, err
	}
	if err := validateBoard(b); err != nil {
		return nil, err
	}
	normalize(b)
	if err := s.save(b); err != nil {
		return nil, err
	}
	return b, nil
}

// Canonicalize rewrites one board in the form a save writes, under the same
// locks every other writer takes, and reports whether the bytes changed. It
// is how `check --fix` repairs layout_not_canonical: the repair planner may
// have read the file a moment ago, but a canvas can save in that moment, so
// the bytes written are rendered from a read taken under the lock and never
// from the planner's copy. A file that no longer parses is left alone and
// reported, since the planner's finding is stale and the next check will say
// what is wrong now.
func (s *Store) Canonicalize(board string) (bool, error) {
	unlock, err := s.lock()
	if err != nil {
		return false, err
	}
	defer unlock()
	p, err := s.path(board)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return false, err
	}
	b, err := Parse(board, data)
	if err != nil {
		return false, err
	}
	normalize(b)
	if bytes.Equal(data, render(b)) {
		return false, nil
	}
	return true, s.save(b)
}

// ProblemKind says what is wrong with a board file. The kinds map one to one
// onto the finding codes of plan 11, but the codes are ticket's to publish, so
// this package names the condition and ticket names the code.
type ProblemKind int

const (
	// Invalid is a file under the canvas directory that is not a board: its
	// name is outside the board grammar, or it does not parse or validate.
	Invalid ProblemKind = iota
	// TicketMissing is a card or a frame member naming a ticket the store does
	// not have. Field says which record.
	TicketMissing
	// LabelUnknown is a pen label outside the store's allowlist.
	LabelUnknown
	// NotCanonical is a valid file whose bytes are not what render would
	// write. Canonical carries what it should be, so a repair is a rewrite.
	NotCanonical
)

// Problem is one thing Check has to say about one board file.
type Problem struct {
	Kind ProblemKind
	// File is relative to the store, with forward slashes: canvas/<name>.yml.
	File string
	// Field names the record the problem is about, in the file's own terms:
	// cards.ID, frames.ID.members, pens.ID.requiredLabels. Empty when the
	// problem is the whole file.
	Field   string
	Message string
	// Canonical is set for NotCanonical: the bytes the file should hold.
	Canonical []byte
}

// Check reads every board under storePath/canvas and reports what is wrong.
// exists says whether a ticket ID is in the store, and knownLabel whether a
// label is in its allowlist; both are ticket's to answer, which is why they
// are passed in rather than read here. A store with no canvas directory has
// nothing to report.
//
// Only .yml files are boards. Anything else in the directory is left alone,
// the way a stray file beside the tickets is: a save in flight leaves a
// dot-prefixed temporary there for a moment, and reporting one would be noise
// about a race nobody can act on.
//
// Problems come back sorted by file and then field, so two runs over one
// store compare directly.
func Check(storePath string, exists func(id string) bool, knownLabel func(label string) bool) ([]Problem, error) {
	dir := filepath.Join(storePath, DirName)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Problem
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		rel := DirName + "/" + e.Name()
		name := strings.TrimSuffix(e.Name(), ".yml")
		if !boardNameOK(name) {
			out = append(out, Problem{Kind: Invalid, File: rel,
				Message: fmt.Sprintf("%q is not a board name: letters, digits, - and _ only", name)})
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		b, err := Parse(name, data)
		if err != nil {
			out = append(out, Problem{Kind: Invalid, File: rel, Message: err.Error()})
			continue
		}
		out = append(out, checkBoard(rel, data, b, exists, knownLabel)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Field < out[j].Field
	})
	return out, nil
}

func checkBoard(rel string, data []byte, b *Board, exists func(string) bool, knownLabel func(string) bool) []Problem {
	var out []Problem
	for id := range b.Cards {
		if !exists(id) {
			out = append(out, Problem{Kind: TicketMissing, File: rel, Field: "cards." + id,
				Message: fmt.Sprintf("card %s is placed but no ticket has that ID", id)})
		}
	}
	for id, f := range b.Frames {
		for _, member := range f.Members {
			if !exists(member) {
				out = append(out, Problem{Kind: TicketMissing, File: rel, Field: "frames." + id + ".members",
					Message: fmt.Sprintf("frame %s lists %s but no ticket has that ID", id, member)})
			}
		}
	}
	for id, p := range b.Pens {
		for _, label := range p.RequiredLabels {
			if !knownLabel(label) {
				out = append(out, Problem{Kind: LabelUnknown, File: rel, Field: "pens." + id + ".requiredLabels",
					Message: fmt.Sprintf("pen %s requires %q, which is not in the config.yml allowlist", id, label)})
			}
		}
	}
	// A parsed board keeps its coordinates as written, so the comparison is
	// against what a save would write from this exact board: rounded, sorted,
	// deduplicated, at the current schema.
	want := *b
	want.Cards = make(map[string]Card, len(b.Cards))
	for id, c := range b.Cards {
		want.Cards[id] = c
	}
	want.Frames = make(map[string]Frame, len(b.Frames))
	for id, f := range b.Frames {
		want.Frames[id] = f
	}
	normalize(&want)
	if canonical := render(&want); !bytes.Equal(data, canonical) {
		out = append(out, Problem{Kind: NotCanonical, File: rel, Canonical: canonical,
			Message: "the file is not in the form a save writes; run check --fix to rewrite it"})
	}
	return out
}
