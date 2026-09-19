package layout

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/terva-sh/git-ticket/ticket"
	"gopkg.in/yaml.v3"
)

// Frame holds a boundary and explicit membership, independent of card placement.
type Frame struct {
	Title   string   `yaml:"title" json:"title"`
	X       float64  `yaml:"x" json:"x"`
	Y       float64  `yaml:"y" json:"y"`
	W       float64  `yaml:"w" json:"w"`
	H       float64  `yaml:"h" json:"h"`
	Color   string   `yaml:"color" json:"color"`
	Members []string `yaml:"members" json:"members"`
}

// Expectations is a record-level read set. Nil means the record must be absent.
type Expectations struct {
	Routing *Routing          `json:"routing"`
	Cards   map[string]*Card  `json:"cards"`
	Frames  map[string]*Frame `json:"frames"`
}

var ErrConflict = errors.New("layout conflict")

// Empty always exposes maps, including for schema 1 layouts and unsaved boards.
func Empty(name string) *Board {
	return &Board{Schema: Schema, Board: name, Cards: map[string]Card{}, Frames: map[string]Frame{}, Routing: emptyRouting()}
}

// Parse is shared by disk loads and the live snapshot's captured file image.
// Unknown fields and versions refuse writes rather than losing future metadata.
func Parse(name string, data []byte) (*Board, error) {
	var raw Board
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, errors.New("layout must contain one YAML document")
	}
	if raw.Schema < 1 || raw.Schema > Schema {
		return nil, fmt.Errorf("unsupported layout schema %d", raw.Schema)
	}
	if raw.Board != name {
		return nil, errors.New("layout board name does not match filename")
	}
	if raw.Schema == 1 && len(raw.Frames) != 0 {
		return nil, errors.New("frames require layout schema 2")
	}
	if raw.Schema < 3 {
		// Even null/empty routing fields belong to schema 3.
		var keys map[string]any
		if err := yaml.Unmarshal(data, &keys); err != nil {
			return nil, err
		}
		for _, key := range []string{"pens", "ruleOrder", "inbox"} {
			if _, exists := keys[key]; exists {
				return nil, errors.New("routing requires layout schema 3")
			}
		}
		raw.Routing = emptyRouting()
	}
	raw.Schema = Schema
	if raw.Cards == nil {
		raw.Cards = map[string]Card{}
	}
	if raw.Frames == nil {
		raw.Frames = map[string]Frame{}
	}
	if err := validateBoard(&raw); err != nil {
		return nil, err
	}
	// Loading never changes authored coordinates or creates manual placements.
	for id, f := range raw.Frames {
		f.Members = append([]string{}, f.Members...)
		sort.Strings(f.Members)
		raw.Frames[id] = f
	}
	raw.Routing = canonicalRouting(raw.Routing, false)
	return &raw, nil
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) && math.Abs(n) <= 1e9 }
func validRecordID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
func validateBoard(b *Board) error {
	if b.Schema < 1 || b.Schema > Schema {
		return fmt.Errorf("unsupported layout schema %d", b.Schema)
	}
	if !boardNameOK(b.Board) {
		return errors.New("invalid board name")
	}
	for id, c := range b.Cards {
		if !validRecordID(id) || !finite(c.X) || !finite(c.Y) || !finite(c.W) || c.W < 0 {
			return fmt.Errorf("invalid card %q", id)
		}
	}
	owners := map[string]string{}
	for id, f := range b.Frames {
		if !validRecordID(id) {
			return fmt.Errorf("invalid frame ID %q", id)
		}
		if strings.TrimSpace(f.Title) == "" || !utf8.ValidString(f.Title) || utf8.RuneCountInString(f.Title) > 80 || strings.ContainsFunc(f.Title, unicode.IsControl) {
			return fmt.Errorf("invalid frame title for %s", id)
		}
		if !finite(f.X) || !finite(f.Y) || !finite(f.W) || !finite(f.H) || round2(f.W) <= 0 || round2(f.H) <= 0 {
			return fmt.Errorf("invalid frame geometry for %s", id)
		}
		switch f.Color {
		case "#759bcc", "#b499be", "#89ad97":
		default:
			return fmt.Errorf("invalid frame color for %s", id)
		}
		for _, member := range f.Members {
			if !ticket.ValidID(member) {
				return fmt.Errorf("invalid frame member %q", member)
			}
			if previous, ok := owners[member]; ok {
				return fmt.Errorf("member %s assigned more than once in %s and %s", member, previous, id)
			}
			owners[member] = id
		}
	}
	return validateRouting(b.Routing)
}

func normalize(b *Board) {
	b.Routing = canonicalRouting(b.Routing, true)
	for id, c := range b.Cards {
		c.X, c.Y, c.W = round2(c.X), round2(c.Y), round2(c.W)
		b.Cards[id] = c
	}
	for id, f := range b.Frames {
		f.X, f.Y, f.W, f.H = round2(f.X), round2(f.Y), round2(f.W), round2(f.H)
		f.Members = append([]string{}, f.Members...)
		sort.Strings(f.Members)
		b.Frames[id] = f
	}
}
func sameFrame(a, b Frame) bool {
	am, bm := slices.Clone(a.Members), slices.Clone(b.Members)
	sort.Strings(am)
	sort.Strings(bm)
	return a.Title == b.Title && a.X == b.X && a.Y == b.Y && a.W == b.W && a.H == b.H && a.Color == b.Color && slices.Equal(am, bm)
}

// Transaction reloads, checks every expectation, applies both maps and renames
// one file under the same lock used by sparse card writes. validate runs after
// conflicts, before saving, so the API can check current ticket identities.
func (s *Store) Transaction(board string, cards map[string]*Card, frames map[string]*Frame, expect *Expectations, validate func() error) (*Board, error) {
	return s.RoutingTransaction(board, cards, frames, nil, expect, validate)
}

// RoutingTransaction includes an optional whole-routing replacement in the
// same read set and atomic write as card and frame edits.
func (s *Store) RoutingTransaction(board string, cards map[string]*Card, frames map[string]*Frame, routing *Routing, expect *Expectations, validate func() error) (*Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expect == nil {
		return nil, errors.New("layout transactions require expect")
	}
	if routing != nil {
		if err := validateRouting(*routing); err != nil {
			return nil, err
		}
		if expect.Routing == nil {
			return nil, errors.New("missing routing expectation")
		}
	}
	if expect.Routing != nil {
		if err := validateRouting(*expect.Routing); err != nil {
			return nil, fmt.Errorf("invalid routing expectation: %w", err)
		}
	}
	for id := range cards {
		if _, ok := expect.Cards[id]; !ok {
			return nil, fmt.Errorf("missing card expectation for %s", id)
		}
	}
	for id := range frames {
		if _, ok := expect.Frames[id]; !ok {
			return nil, fmt.Errorf("missing frame expectation for %s", id)
		}
	}
	b, err := s.Load(board)
	if err != nil {
		return nil, err
	}
	if expect.Routing != nil && !sameRouting(b.Routing, canonicalRouting(*expect.Routing, false)) {
		return nil, fmt.Errorf("%w: routing changed", ErrConflict)
	}
	for id, expected := range expect.Cards {
		current, ok := b.Cards[id]
		if expected == nil && ok || expected != nil && (!ok || current != *expected) {
			return nil, fmt.Errorf("%w: card %s changed", ErrConflict, id)
		}
	}
	for id, expected := range expect.Frames {
		current, ok := b.Frames[id]
		if expected == nil && ok || expected != nil && (!ok || !sameFrame(current, *expected)) {
			return nil, fmt.Errorf("%w: frame %s changed", ErrConflict, id)
		}
	}
	if routing != nil {
		b.Routing = canonicalRouting(*routing, false)
	}
	for id, c := range cards {
		if c == nil {
			delete(b.Cards, id)
		} else {
			b.Cards[id] = *c
		}
	}
	for id, f := range frames {
		if f == nil {
			delete(b.Frames, id)
		} else {
			b.Frames[id] = *f
		}
	}
	if err := validateBoard(b); err != nil {
		return nil, err
	}
	if validate != nil {
		if err := validate(); err != nil {
			return nil, err
		}
	}
	normalize(b)
	if err := s.save(b); err != nil {
		return nil, err
	}
	return b, nil
}

// RemoveTicket is deletion cleanup, not undo of manual placement. A nil card in
// Update or Transaction deliberately leaves membership alone.
func (s *Store) RemoveTicket(board, id string) (*Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.Load(board)
	if err != nil {
		return nil, err
	}
	delete(b.Cards, id)
	for key, f := range b.Frames {
		f.Members = slices.DeleteFunc(f.Members, func(member string) bool { return member == id })
		b.Frames[key] = f
	}
	if err := s.save(b); err != nil {
		return nil, err
	}
	return b, nil
}

func renderFrames(sb *strings.Builder, frames map[string]Frame) {
	if len(frames) == 0 {
		sb.WriteString("frames: {}\n")
		return
	}
	ids := make([]string, 0, len(frames))
	for id := range frames {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	sb.WriteString("frames:\n")
	for _, id := range ids {
		f := frames[id]
		members := append([]string{}, f.Members...)
		sort.Strings(members)
		fmt.Fprintf(sb, "  %s: {title: %s, x: %s, y: %s, w: %s, h: %s, color: %s, members: [%s]}\n", strconv.Quote(id), strconv.Quote(f.Title), num(f.X), num(f.Y), num(f.W), num(f.H), strconv.Quote(f.Color), strings.Join(members, ", "))
	}
}
