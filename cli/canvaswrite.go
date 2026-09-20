package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/terva-sh/git-ticket/layout"
	"github.com/terva-sh/git-ticket/ticket"
)

const canvasWords = "show, pens, explain ID, pen add ID, pen rm ID, pen order ID..., place ID, release ID..., frame add ID, or inbox"

// canvasFlags is every flag a canvas word can take, registered once because
// the words share one FlagSet. Each word names the ones it reads, and a flag
// set for a word that does not read it is a usage error rather than silently
// ignored: `place ID --title T` is a mistake the caller wants told about.
type canvasFlags struct {
	title   string
	labels  stringList
	at      string
	size    string
	color   string
	pin     string
	members stringList
}

// set names the flags the caller gave. A flag is given when it holds
// something, which is the same test every word applies before reading one.
func (f canvasFlags) set() []string {
	var out []string
	if f.title != "" {
		out = append(out, "--title")
	}
	if len(f.labels) > 0 {
		out = append(out, "--label")
	}
	if f.at != "" {
		out = append(out, "--at")
	}
	if f.size != "" {
		out = append(out, "--size")
	}
	if f.color != "" {
		out = append(out, "--color")
	}
	if f.pin != "" {
		out = append(out, "--pin")
	}
	if len(f.members) > 0 {
		out = append(out, "--member")
	}
	return out
}

// only refuses a flag the word does not read. It returns the flags as an
// error so the caller can phrase the refusal in its own words.
func (f canvasFlags) only(allowed ...string) error {
	var extra []string
	for _, name := range f.set() {
		if !slices.Contains(allowed, name) {
			extra = append(extra, name)
		}
	}
	if len(extra) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(extra, ", "))
}

func (f canvasFlags) none() error { return f.only() }

// point parses X,Y. A coordinate is what the caller chose, per 12.10, so the
// only checks are that there are two of them and each is a number.
func point(flagName, v string) (*layout.Point, error) {
	x, y, err := pair(v)
	if err != nil {
		return nil, usageErr("%s takes X,Y, two numbers separated by a comma: %v", flagName, err)
	}
	return &layout.Point{X: x, Y: y}, nil
}

func pair(v string) (float64, float64, error) {
	parts := strings.Split(v, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("got %q", v)
	}
	a, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q is not a number", parts[0])
	}
	b, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q is not a number", parts[1])
	}
	return a, b, nil
}

// refusal is a write the layout package would not accept, or one the word
// itself declines, reported under validation_failed so a caller reading the
// envelope sees the same code a refused ticket write carries.
func refusal(format string, args ...any) error {
	return &ticket.Error{Code: ticket.CodeValidationFailed, Message: fmt.Sprintf(format, args...)}
}

// runCanvasWrite dispatches the write words. Every one resolves its ticket
// IDs through the store first, so a prefix works and a typo is ticket_not_found
// before anything is written, then hands one function to layout.Modify.
func runCanvasWrite(ctx *cmdContext, board string, f canvasFlags, rest []string) error {
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	ls := layout.New(s.Path())
	var edit func(*layout.Board) error
	var said string
	// warn is said after the write lands and not before, so a refused write
	// never claims that check will warn about something never written.
	var warn []string

	word := rest[0]
	switch word {
	case "pen":
		if len(rest) < 2 {
			return usageErr("canvas pen takes a word: add ID, rm ID, or order ID...")
		}
		switch rest[1] {
		case "add":
			if len(rest) != 3 {
				return usageErr("canvas pen add takes one pen ID")
			}
			id := rest[2]
			if err := f.only("--title", "--label", "--at", "--size", "--color", "--pin"); err != nil {
				return usageErr("canvas pen add does not take %s", err)
			}
			pen, err := f.pen()
			if err != nil {
				return err
			}
			// The allowlist is advisory, per plan 11: create files a ticket
			// under a label nobody listed and check warns, and a rule that
			// names one gets the same treatment, because a pen is often
			// written for a label that is about to exist. The warning goes
			// where create's would, so a caller under --json still sees it.
			for _, label := range pen.RequiredLabels {
				if !s.Config().KnownLabel(label) {
					warn = append(warn, fmt.Sprintf("pen %s requires %q, which is not in the config.yml allowlist; check will warn until it is", id, label))
				}
			}
			edit = func(b *layout.Board) error {
				if _, ok := b.Pens[id]; ok {
					return refusal("board %s already has a pen %s; rm it first, or pick another ID", b.Board, id)
				}
				b.Pens[id] = pen
				b.RuleOrder = append(b.RuleOrder, id)
				return nil
			}
			said = fmt.Sprintf("pen %s added to board %s as its last rule", id, board)
		case "rm":
			if len(rest) != 3 {
				return usageErr("canvas pen rm takes one pen ID")
			}
			id := rest[2]
			if err := f.none(); err != nil {
				return usageErr("canvas pen rm does not take %s", err)
			}
			edit = func(b *layout.Board) error {
				if _, ok := b.Pens[id]; !ok {
					return refusal("board %s has no pen %s", b.Board, id)
				}
				delete(b.Pens, id)
				b.RuleOrder = slices.DeleteFunc(b.RuleOrder, func(r string) bool { return r == id })
				return nil
			}
			said = fmt.Sprintf("pen %s removed from board %s; its cards go by the rules that remain", id, board)
		case "order":
			ids := rest[2:]
			if len(ids) == 0 {
				return usageErr("canvas pen order takes every pen ID, in the order the rules should resolve")
			}
			if err := f.none(); err != nil {
				return usageErr("canvas pen order does not take %s", err)
			}
			edit = func(b *layout.Board) error {
				if err := sameSet(ids, b.Pens); err != nil {
					return refusal("pen order must name every pen on board %s exactly once: %v", b.Board, err)
				}
				b.RuleOrder = append([]string{}, ids...)
				return nil
			}
			said = fmt.Sprintf("board %s resolves in the order %s", board, strings.Join(ids, ", "))
		default:
			return usageErr("%q is not a canvas pen word; use add ID, rm ID, or order ID...", rest[1])
		}
	case "place":
		if len(rest) != 2 {
			return usageErr("canvas place takes one ticket ID")
		}
		if err := f.only("--at"); err != nil {
			return usageErr("canvas place does not take %s", err)
		}
		if f.at == "" {
			return usageErr("canvas place needs --at X,Y, the position the card is pinned at")
		}
		at, err := point("--at", f.at)
		if err != nil {
			return err
		}
		t, err := s.Get(context.Background(), rest[1])
		if err != nil {
			return err
		}
		edit = func(b *layout.Board) error {
			c := b.Cards[t.ID]
			c.X, c.Y = at.X, at.Y
			b.Cards[t.ID] = c
			return nil
		}
		said = fmt.Sprintf("%s pinned at (%s, %s) on board %s; routing does not apply until it is released", t.ID, num(at.X), num(at.Y), board)
	case "release":
		refs := rest[1:]
		if len(refs) == 0 {
			return usageErr("canvas release takes one or more ticket IDs")
		}
		if err := f.none(); err != nil {
			return usageErr("canvas release does not take %s", err)
		}
		ids := make([]string, 0, len(refs))
		for _, ref := range refs {
			t, err := s.Get(context.Background(), ref)
			if err != nil {
				return err
			}
			ids = append(ids, t.ID)
		}
		edit = func(b *layout.Board) error {
			for _, id := range ids {
				if _, ok := b.Cards[id]; !ok {
					return refusal("%s is not pinned on board %s; the rules already place it", id, b.Board)
				}
				delete(b.Cards, id)
			}
			return nil
		}
		said = fmt.Sprintf("released %s on board %s; the rules place them now", strings.Join(ids, ", "), board)
	case "frame":
		if len(rest) != 3 || rest[1] != "add" {
			return usageErr("canvas frame takes add ID")
		}
		id := rest[2]
		if err := f.only("--title", "--at", "--size", "--color", "--member"); err != nil {
			return usageErr("canvas frame add does not take %s", err)
		}
		frame, err := f.frame()
		if err != nil {
			return err
		}
		members := make([]string, 0, len(f.members))
		for _, ref := range f.members {
			t, err := s.Get(context.Background(), ref)
			if err != nil {
				return err
			}
			members = append(members, t.ID)
		}
		frame.Members = members
		edit = func(b *layout.Board) error {
			if _, ok := b.Frames[id]; ok {
				return refusal("board %s already has a frame %s", b.Board, id)
			}
			b.Frames[id] = frame
			return nil
		}
		said = fmt.Sprintf("frame %s added to board %s with %d members", id, board, len(members))
	case "inbox":
		if len(rest) != 1 {
			return usageErr("canvas inbox takes no arguments")
		}
		if err := f.only("--at"); err != nil {
			return usageErr("canvas inbox does not take %s", err)
		}
		if f.at == "" {
			return usageErr("canvas inbox needs --at X,Y, where unmatched cards land")
		}
		at, err := point("--at", f.at)
		if err != nil {
			return err
		}
		edit = func(b *layout.Board) error {
			b.Inbox = at
			return nil
		}
		said = fmt.Sprintf("inbox of board %s at (%s, %s)", board, num(at.X), num(at.Y))
	}

	// The JSON answer lists what each pen catches, which needs every ticket.
	// They are read before the write so that nothing fallible runs between a
	// committed rename and the report of it: a caller told the write failed
	// after it landed would retry into a duplicate. A write changes no ticket,
	// so the listing is as current afterwards as it was before.
	var tickets []*ticket.Ticket
	if ctx.g.json {
		if tickets, err = s.List(context.Background(), ticket.Filter{All: true}); err != nil {
			return err
		}
	}
	b, err := ls.Modify(board, edit)
	if err != nil {
		var te *ticket.Error
		if errors.As(err, &te) {
			return err
		}
		if errors.Is(err, layout.ErrLockTimeout) {
			return &ticket.Error{Code: ticket.CodeLockTimeout, Message: "another process is writing the canvas directory: " + err.Error(), Err: err}
		}
		return refusal("board %s refused: %v", board, err)
	}
	for _, w := range warn {
		fmt.Fprintf(ctx.env.Stderr, "git-ticket: %s\n", w)
	}
	view := &boardView{board: b, path: displayPath(s, filepath.Join(ls.Dir(), board+".yml")), exists: true}
	if ctx.g.json {
		writeJSON(ctx.out, newCanvasBoardEnvelope(view, summarize(view, tickets)))
		return nil
	}
	fmt.Fprintf(ctx.out, "%s\n  %s\n", said, view.path)
	return nil
}

// pen builds the record pen add writes. The title, at least one label, the
// origin and the size are required because a pen without any of them is not
// one the canvas can draw or resolve. The colour defaults to the first of the
// three the canvas draws, and the pin to the origin: the reference resolver
// reads no pin, per 12.10, so a person who wants one names it.
func (f canvasFlags) pen() (layout.Pen, error) {
	var p layout.Pen
	if strings.TrimSpace(f.title) == "" {
		return p, usageErr("canvas pen add needs --title")
	}
	if len(f.labels) == 0 {
		return p, usageErr("canvas pen add needs at least one --label; the pen catches a ticket carrying all of them")
	}
	if f.at == "" || f.size == "" {
		return p, usageErr("canvas pen add needs --at X,Y and --size W,H, the region the pen fills")
	}
	at, err := point("--at", f.at)
	if err != nil {
		return p, err
	}
	w, h, err := pair(f.size)
	if err != nil {
		return p, usageErr("--size takes W,H, two numbers separated by a comma: %v", err)
	}
	pin := at
	if f.pin != "" {
		if pin, err = point("--pin", f.pin); err != nil {
			return p, err
		}
	}
	color := f.color
	if color == "" {
		color = layout.Colors[0]
	}
	return layout.Pen{Title: f.title, X: at.X, Y: at.Y, W: w, H: h, Color: color, Pin: pin, RequiredLabels: append([]string{}, f.labels...)}, nil
}

func (f canvasFlags) frame() (layout.Frame, error) {
	var fr layout.Frame
	if strings.TrimSpace(f.title) == "" {
		return fr, usageErr("canvas frame add needs --title")
	}
	if f.at == "" || f.size == "" {
		return fr, usageErr("canvas frame add needs --at X,Y and --size W,H, the region the frame bounds")
	}
	at, err := point("--at", f.at)
	if err != nil {
		return fr, err
	}
	w, h, err := pair(f.size)
	if err != nil {
		return fr, usageErr("--size takes W,H, two numbers separated by a comma: %v", err)
	}
	color := f.color
	if color == "" {
		color = layout.Colors[0]
	}
	return layout.Frame{Title: f.title, X: at.X, Y: at.Y, W: w, H: h, Color: color}, nil
}

// sameSet says how ids differs from the pens on the board, in words a person
// can act on: what was named twice, what was named and does not exist, and
// what exists and was left out.
func sameSet(ids []string, pens map[string]layout.Pen) error {
	seen := map[string]bool{}
	var twice, unknown, missing []string
	for _, id := range ids {
		if seen[id] {
			twice = append(twice, id)
		}
		seen[id] = true
		if _, ok := pens[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	for id := range pens {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	slices.Sort(missing)
	var parts []string
	if len(twice) > 0 {
		parts = append(parts, "named twice: "+strings.Join(twice, ", "))
	}
	if len(unknown) > 0 {
		parts = append(parts, "no such pen: "+strings.Join(unknown, ", "))
	}
	if len(missing) > 0 {
		parts = append(parts, "left out: "+strings.Join(missing, ", "))
	}
	if len(parts) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}
