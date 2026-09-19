// Package layout stores manual card placements and grouping frames on a canvas.
//
// The file it defines is .tickets/canvas/<board>.yml, written by
// git-ticket-canvas and read by nothing else yet. It lives in this module
// because the store's format is this tool's to define, and check and the CLI
// cannot reach a package that is internal to the viewer. Copied unchanged from
// git-ticket-canvas at 9d1f6ed2122300747dc5ad10075830f5927f9bb6; that
// repository's history holds why each line is the way it is.
//
// Position is authored data, not derived data: the whole reason a canvas beats
// a list is that you remember where you put things, so an arrangement has to
// survive a cache rebuild the same way a ticket survives one. That rules out
// keeping it only in an application database, and it rules out a binary file
// in the repository, because git-ticket's format is per-field mergeable text
// and a SQLite page is the one file in the store that could not be merged.
//
// A board is text, one line per card or frame, sorted by stable ID. Dragging a
// card is a one-line diff. The file holds positions, frame boundaries, and frame
// membership. Ticket content stays in ticket files.
//
// Derived state — a search index, viewport, presence, undo — is the part that
// wants a real database, and it belongs in a gitignored cache beside this,
// rebuilt from disk. Nothing here writes to that cache, because nothing here
// is allowed to be the thing that goes stale.
package layout

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Schema is the layout file version this package reads and writes.
const Schema = 3

// DirName is the directory boards live in, inside the ticket store.
const DirName = "canvas"

// DefaultBoard is the board a client gets when it names none.
const DefaultBoard = "default"

// Card is where one ticket sits. Width is stored because a card holding a long
// title is worth widening and that choice is as much a placement as x and y.
// Height is not: it follows the content, and storing it would let a stale
// number fight the renderer.
type Card struct {
	X         float64 `yaml:"x" json:"x"`
	Y         float64 `yaml:"y" json:"y"`
	W         float64 `yaml:"w,omitempty" json:"w,omitempty"`
	Z         int     `yaml:"z,omitempty" json:"z,omitempty"`
	Collapsed bool    `yaml:"collapsed,omitempty" json:"collapsed,omitempty"`
}

// Board is one canvas with manual card placements and explicit grouping frames.
type Board struct {
	Routing `yaml:",inline"`
	Schema  int              `yaml:"schema" json:"schema"`
	Board   string           `yaml:"board" json:"board"`
	Cards   map[string]Card  `yaml:"cards" json:"cards"`
	Frames  map[string]Frame `yaml:"frames" json:"frames"`
}

// Store reads and writes boards under a ticket store's canvas directory.
//
// It serialises its own writes. Two browser tabs dragging at once is the
// ordinary case, and a board is small enough that a mutex costs nothing next
// to the file write it guards.
type Store struct {
	dir string
	mu  sync.Mutex
}

// New returns a Store writing under storePath/canvas, where storePath is the
// .tickets directory of a git-ticket store.
func New(storePath string) *Store {
	return &Store{dir: filepath.Join(storePath, DirName)}
}

// Dir is where boards are written.
func (s *Store) Dir() string { return s.dir }

var boardNameOK = func(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func (s *Store) path(board string) (string, error) {
	if !boardNameOK(board) {
		return "", fmt.Errorf("invalid board name %q: letters, digits, - and _ only", board)
	}
	return filepath.Join(s.dir, board+".yml"), nil
}

// Load reads a board. A board that does not exist yet is not an error: it is
// an empty board, because the first card placed on a canvas should not have to
// be preceded by a command that creates the file.
func (s *Store) Load(board string) (*Board, error) {
	b, _, err := s.Read(board)
	return b, err
}

// Read is Load and whether a file was read, from the one read. A caller that
// reports both, the way `git ticket canvas` does, must not learn the second
// from a separate stat: a save is a rename, and between a read and a stat the
// file can appear or go, leaving a populated board reported as absent or an
// empty one as present.
func (s *Store) Read(board string) (*Board, bool, error) {
	p, err := s.path(board)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return Empty(board), false, nil
	}
	if err != nil {
		return nil, false, err
	}
	b, err := Parse(board, data)
	if err != nil {
		return nil, true, err
	}
	return b, true, nil
}

// Save writes a board, replacing what was there.
func (s *Store) Save(b *Board) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Load(b.Board); err != nil {
		return err
	}
	return s.save(b)
}

func (s *Store) save(b *Board) error {
	// Preserve source compatibility for callers constructing card-only boards.
	// Parsed schema-3 files must supply routing explicitly.
	copy := *b
	b = &copy
	if b.Pens == nil && b.RuleOrder == nil && b.Inbox == nil {
		b.Routing = emptyRouting()
	}
	if err := validateBoard(b); err != nil {
		return err
	}
	b.Routing = canonicalRouting(b.Routing, true)
	p, err := s.path(b.Board)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, "."+b.Board+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(render(b)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

// Update applies a set of card placements to a board and writes it. A card
// mapped to nil loses its manual placement. This does not delete the ticket
// or its frame membership.
//
// It reloads under the lock rather than taking a board from the caller, so a
// drag in one tab does not overwrite a drag in another with a stale copy of
// every other card.
func (s *Store) Update(board string, cards map[string]*Card) (*Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.Load(board)
	if err != nil {
		return nil, err
	}
	for id, c := range cards {
		if c == nil {
			delete(b.Cards, id)
			continue
		}
		b.Cards[id] = *c
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

// Boards lists the board names that exist.
func (s *Store) Boards() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return []string{DefaultBoard}, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".yml"))
	}
	if len(names) == 0 {
		names = []string{DefaultBoard}
	}
	sort.Strings(names)
	return names, nil
}

// render emits the canonical bytes for a board: a fixed header, then one line
// per card sorted by ID.
//
// It is hand-emitted rather than handed to yaml.Marshal for the reason
// git-ticket renders its own tickets: the diff is the interface. A marshaller
// free to choose block style would turn one drag into six changed lines, and
// free to choose map order would reorder the file whenever Go's map iteration
// felt like it. Both are the difference between a board that merges and one
// that conflicts.
func render(b *Board) []byte {
	ids := make([]string, 0, len(b.Cards))
	for id := range b.Cards {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var sb strings.Builder
	sb.WriteString("# git-ticket canvas layout. Positions and frames; tickets live in their own files.\n")
	sb.WriteString("# One line per card, sorted by ticket ID, so a drag is a one-line diff.\n")
	fmt.Fprintf(&sb, "schema: %d\n", Schema)
	fmt.Fprintf(&sb, "board: %s\n", strconv.Quote(b.Board))
	if len(ids) == 0 {
		sb.WriteString("cards: {}\n")
	} else {
		sb.WriteString("cards:\n")
	}
	for _, id := range ids {
		c := b.Cards[id]
		fmt.Fprintf(&sb, "  %s: {x: %s, y: %s", strconv.Quote(id), num(c.X), num(c.Y))
		if c.W != 0 {
			fmt.Fprintf(&sb, ", w: %s", num(c.W))
		}
		if c.Z != 0 {
			fmt.Fprintf(&sb, ", z: %d", c.Z)
		}
		if c.Collapsed {
			sb.WriteString(", collapsed: true")
		}
		sb.WriteString("}\n")
	}
	renderFrames(&sb, b.Frames)
	renderRouting(&sb, b.Routing)
	return []byte(sb.String())
}

// num formats a coordinate without a trailing ".0", so a whole-pixel position
// reads as an integer and a half-pixel one does not lose its half. Positions
// are rounded to two decimals: a canvas cannot express more, and trailing
// float noise would rewrite lines nothing moved.
func num(f float64) string {
	s := strconv.FormatFloat(round2(f), 'f', -1, 64)
	if s == "-0" {
		return "0"
	}
	return s
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
