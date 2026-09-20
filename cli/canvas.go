package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/terva-sh/git-ticket/layout"
	"github.com/terva-sh/git-ticket/ticket"
)

// runCanvas is the `git ticket canvas` family, per plan 12.10. Three words
// read: show prints each pen with what it catches, pens prints the rules in
// resolution order, and explain says where one card is and why. Seven write:
// pen add, pen rm, pen order, place, release, frame add, and inbox, each
// through one layout.Modify that validates before it renames, so a refused
// write leaves the file as it was. The write words are in canvaswrite.go.
//
// It is one command with words rather than ten commands, the way series is,
// because they all answer one question from one file and a caller that has
// found `canvas` has found all of them.
//
// Nothing here computes a position. place writes the coordinate its caller
// chose and everything else writes rules; where a card lands within a pen is
// the canvas's to compute, and the one thing this file must never grow is a
// second copy of that.
func runCanvas(ctx *cmdContext, args []string) error {
	board := layout.DefaultBoard
	var f canvasFlags
	rest, err := ctx.parseFlags("canvas", args, func(fs *flag.FlagSet) {
		fs.StringVar(&board, "board", board, "the board to read or write")
		fs.Var(&f.title, "title", "the pen's or frame's title")
		fs.Var(&f.labels, "label", "a label the pen requires; repeatable, and the pen requires all of them")
		fs.Var(&f.status, "status", "a status the pen catches; repeatable, and any one of them matches")
		fs.Var(&f.types, "type", "a ticket type the pen catches; repeatable, and any one of them matches")
		fs.Var(&f.parents, "parent", "a parent the pen catches; repeatable, and any one of them matches")
		fs.Var(&f.at, "at", "a position, as X,Y")
		fs.Var(&f.size, "size", "a region's size, as W,H")
		fs.Var(&f.color, "color", "the pen's or frame's colour, one of "+strings.Join(layout.Colors, ", "))
		fs.Var(&f.pin, "pin", "the pen's pin point, as X,Y; the pen's origin when unset")
		fs.Var(&f.members, "member", "a ticket the frame holds; repeatable")
	})
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return usageErr("canvas takes a word: %s", canvasWords)
	}
	word := rest[0]
	switch word {
	case "show", "pens":
		if len(rest) != 1 {
			return usageErr("canvas %s takes no arguments", word)
		}
	case "explain":
		if len(rest) != 2 {
			return usageErr("canvas explain takes one ticket ID")
		}
	case "pen", "place", "release", "frame", "inbox":
		return runCanvasWrite(ctx, board, f, rest)
	default:
		return usageErr("%q is not a canvas word; use %s", word, canvasWords)
	}
	if err := f.none(); err != nil {
		return usageErr("canvas %s reads the board and takes no %s", word, err)
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	view, err := readBoard(s, board)
	if err != nil {
		return err
	}

	if word == "explain" {
		t, err := s.Get(context.Background(), rest[1])
		if err != nil {
			return err
		}
		e := layout.Explain(view.board, ruleTicket(t))
		if ctx.g.json {
			writeJSON(ctx.out, newCanvasExplainEnvelope(view, t, e))
			return nil
		}
		writeCanvasExplain(ctx.out, view, t, e)
		return nil
	}

	// pens on the page reads the rules alone, so it does not list tickets. Its
	// envelope is canvas-board, which carries what each pen catches, so the
	// JSON form does list, per 10.10.
	if word == "pens" && !ctx.g.json {
		writeCanvasPens(ctx.out, view)
		return nil
	}
	tickets, err := s.List(context.Background(), ticket.Filter{All: true})
	if err != nil {
		return err
	}
	summary := summarize(view, tickets)
	if ctx.g.json {
		writeJSON(ctx.out, newCanvasBoardEnvelope(view, summary))
		return nil
	}
	writeCanvasShow(ctx.out, view, summary)
	return nil
}

// boardView is a board and the two facts about its file a reader needs: where
// it is, and whether it exists. Load answers an empty board for a missing file,
// which is right for the canvas and silent for a person asking what is there.
type boardView struct {
	board  *layout.Board
	path   string
	exists bool
}

func readBoard(s *ticket.Store, name string) (*boardView, error) {
	ls := layout.New(s.Path())
	// Read refuses a name outside the board grammar, letters, digits, - and _,
	// before any path is built from it, so the join below never sees a name
	// that could leave the canvas directory. It answers exists from the same
	// read as the board, so the two cannot describe different files.
	b, exists, err := ls.Read(name)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(ls.Dir(), name+".yml")
	return &boardView{board: b, path: displayPath(s, path), exists: exists}, nil
}

// boardSummary is the board read the way show prints it: each pen with the
// automatic cards it catches, the inbox with what fell through, and the pinned
// cards apart from both, because routing does not apply to them.
type boardSummary struct {
	catches map[string][]*ticket.Ticket
	inbox   []*ticket.Ticket
	pinned  []pinnedCard
}

type pinnedCard struct {
	ticket *ticket.Ticket
	card   layout.Card
}

func summarize(v *boardView, tickets []*ticket.Ticket) boardSummary {
	byID := make(map[string]*ticket.Ticket, len(tickets))
	rules := make([]layout.RuleTicket, 0, len(tickets))
	for _, t := range tickets {
		byID[t.ID] = t
		rules = append(rules, ruleTicket(t))
	}
	out := boardSummary{catches: map[string][]*ticket.Ticket{}}
	for _, e := range layout.Route(v.board, rules) {
		t := byID[e.ID]
		if e.Pinned != nil {
			out.pinned = append(out.pinned, pinnedCard{ticket: t, card: *e.Pinned})
			continue
		}
		if e.Inbox() {
			out.inbox = append(out.inbox, t)
			continue
		}
		out.catches[e.Destination] = append(out.catches[e.Destination], t)
	}
	return out
}

// ruleTicket is what routing reads of a ticket, per plan 12.10. A ticket
// with no parent carries the empty string, which matches a rule that does not
// test the field and no rule that does.
func ruleTicket(t *ticket.Ticket) layout.RuleTicket {
	r := layout.RuleTicket{ID: t.ID, Labels: t.Labels, Status: t.Status, Type: t.Type}
	if t.Parent != nil {
		r.Parent = *t.Parent
	}
	return r
}

// ruleText prints a pen's whole rule in words: the fields it tests, in the
// order layout names them, with labels read as all of them and the other
// three as any of them, per plan 12.10.
func ruleText(m layout.Match) string {
	var parts []string
	if len(m.Labels) > 0 {
		parts = append(parts, "labels "+strings.Join(m.Labels, ", "))
	}
	for _, f := range []struct {
		name   string
		values []string
	}{{"status", m.Status}, {"type", m.Type}, {"parent", m.Parent}} {
		if len(f.values) > 0 {
			parts = append(parts, f.name+" "+strings.Join(f.values, " or "))
		}
	}
	return strings.Join(parts, "; ")
}

// failureText says why a rule passed one ticket over, field by field in the
// order the candidate reports them, naming what the ticket had beside what
// the rule wanted. The comparison is the answer, so neither half is left for
// the reader to go and look up.
func failureText(c layout.Candidate, t *ticket.Ticket) string {
	var parts []string
	for _, field := range c.Failed {
		switch field {
		case "labels":
			parts = append(parts, "missing labels "+strings.Join(c.MissingLabels, ", "))
		case "status":
			parts = append(parts, fmt.Sprintf("status is %s, rule wants %s", t.Status, strings.Join(c.Match.Status, " or ")))
		case "type":
			parts = append(parts, fmt.Sprintf("type is %s, rule wants %s", t.Type, strings.Join(c.Match.Type, " or ")))
		case "parent":
			parts = append(parts, fmt.Sprintf("parent is %s, rule wants %s", orNone(t.Parent), strings.Join(c.Match.Parent, " or ")))
		}
	}
	return strings.Join(parts, "; ")
}

func orNone(v *string) string {
	if v == nil || *v == "" {
		return "none"
	}
	return *v
}

func writeCanvasShow(w io.Writer, v *boardView, s boardSummary) {
	if !v.exists {
		writeNoLayout(w, v)
		return
	}
	fmt.Fprintf(w, "board %s  %s\n", v.board.Board, v.path)
	if len(v.board.RuleOrder) == 0 {
		fmt.Fprintln(w, "no pens; every automatic card goes to the inbox")
	}
	for i, id := range v.board.RuleOrder {
		pen := v.board.Pens[id]
		caught := s.catches[id]
		fmt.Fprintf(w, "\n%d  %s  %s  %s  catches %d\n", i+1, id, pen.Title, ruleText(pen.Match), len(caught))
		writeTicketLines(w, caught)
	}
	fmt.Fprintf(w, "\ninbox  %s  catches %d\n", pointText(v.board.Inbox), len(s.inbox))
	writeTicketLines(w, s.inbox)
	if len(s.pinned) > 0 {
		fmt.Fprintf(w, "\npinned  %d, placed by hand; routing does not apply\n", len(s.pinned))
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, p := range s.pinned {
			fmt.Fprintf(tw, "  %s\t(%s, %s)\t%s\n", p.ticket.ID, num(p.card.X), num(p.card.Y), p.ticket.Title)
		}
		tw.Flush()
	}
}

func writeCanvasPens(w io.Writer, v *boardView) {
	if !v.exists {
		writeNoLayout(w, v)
		return
	}
	if len(v.board.RuleOrder) == 0 {
		fmt.Fprintf(w, "board %s has no pens\n", v.board.Board)
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for i, id := range v.board.RuleOrder {
		pen := v.board.Pens[id]
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", i+1, id, pen.Title, ruleText(pen.Match))
	}
	tw.Flush()
}

func writeCanvasExplain(w io.Writer, v *boardView, t *ticket.Ticket, e layout.Explanation) {
	fmt.Fprintf(w, "%s  %s\n", t.ID, t.Title)
	if !v.exists {
		fmt.Fprintf(w, "board %s has no layout file at %s\n", v.board.Board, v.path)
		fmt.Fprintln(w, "automatic: the canvas places it in status lanes")
		return
	}
	if e.Pinned != nil {
		fmt.Fprintf(w, "pinned at (%s, %s); routing does not apply\n", num(e.Pinned.X), num(e.Pinned.Y))
	} else if len(e.Candidates) == 0 {
		fmt.Fprintln(w, "automatic: the board has no pens, so the canvas places it in status lanes")
	} else {
		fmt.Fprintln(w, "automatic: the canvas places it by the rules below")
	}
	if len(e.Candidates) == 0 {
		return
	}
	fmt.Fprintln(w, "\nrouting:")
	if e.Inbox() {
		fmt.Fprintf(w, "  goes to the inbox %s: no rule matched\n", pointText(v.board.Inbox))
	} else {
		pen := v.board.Pens[e.Destination]
		fmt.Fprintf(w, "  goes to pen %s (%s): %s\n", e.Destination, pen.Title, ruleText(pen.Match))
	}
	for _, c := range e.Candidates {
		switch c.Outcome {
		case layout.NoMatch:
			fmt.Fprintf(w, "  not %s (rule %d): %s\n", c.Pen, c.Order+1, failureText(c, t))
		case layout.LaterRule:
			fmt.Fprintf(w, "  not %s (rule %d): matches, but an earlier rule took it\n", c.Pen, c.Order+1)
		}
	}
}

func writeNoLayout(w io.Writer, v *boardView) {
	fmt.Fprintf(w, "board %s has no layout file at %s; every card is automatic and the canvas places it in status lanes\n", v.board.Board, v.path)
}

func writeTicketLines(w io.Writer, ts []*ticket.Ticket) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, t := range ts {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", t.ID, t.Status, t.Title)
	}
	tw.Flush()
}

func pointText(p *layout.Point) string {
	if p == nil {
		return "(unset)"
	}
	return fmt.Sprintf("(%s, %s)", num(p.X), num(p.Y))
}

// num prints a coordinate the way the board file does: an integer when it is
// one, so a reader compares like with like.
func num(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}

// --- JSON, per plan 10.10 ---------------------------------------------------

type canvasPenJSON struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
	Title string `json:"title"`
	// Match is the pen's whole rule, per plan 10.10, and it carries all four
	// fields as arrays whatever the board file spells.
	Match   layout.Match `json:"match"`
	Tickets []string     `json:"tickets"`
}

type canvasPointJSON struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type canvasInboxJSON struct {
	At      *canvasPointJSON `json:"at"`
	Tickets []string         `json:"tickets"`
}

type canvasPinnedJSON struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type canvasBoardEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Board         string `json:"board"`
	Path          string `json:"path"`
	// Exists is false for a board with no layout file. Every other field is
	// then what an empty board answers, which is what the canvas would show.
	Exists bool `json:"exists"`
	// Applied says the canvas places cards by these rules, which it has since
	// git-ticket-canvas v0.5.0. It was false before v0.22.0 of this tool, so a
	// consumer reads the fact rather than a release number.
	Applied bool               `json:"applied"`
	Pens    []canvasPenJSON    `json:"pens"`
	Inbox   canvasInboxJSON    `json:"inbox"`
	Pinned  []canvasPinnedJSON `json:"pinned"`
}

func newCanvasBoardEnvelope(v *boardView, s boardSummary) canvasBoardEnvelope {
	env := canvasBoardEnvelope{SchemaVersion: schemaVersion, Kind: "canvas-board", Board: v.board.Board, Path: v.path, Exists: v.exists, Applied: true,
		Pens: []canvasPenJSON{}, Inbox: canvasInboxJSON{Tickets: ids(s.inbox)}, Pinned: []canvasPinnedJSON{}}
	for i, id := range v.board.RuleOrder {
		pen := v.board.Pens[id]
		env.Pens = append(env.Pens, canvasPenJSON{ID: id, Order: i, Title: pen.Title, Match: pen.Match, Tickets: ids(s.catches[id])})
	}
	if p := v.board.Inbox; p != nil {
		env.Inbox.At = &canvasPointJSON{X: p.X, Y: p.Y}
	}
	for _, p := range s.pinned {
		env.Pinned = append(env.Pinned, canvasPinnedJSON{ID: p.ticket.ID, X: p.card.X, Y: p.card.Y})
	}
	sort.Slice(env.Pinned, func(i, j int) bool { return env.Pinned[i].ID < env.Pinned[j].ID })
	return env
}

type canvasRoutingJSON struct {
	// Destination is the winning pen's id, or null for the inbox.
	Destination *string            `json:"destination"`
	Candidates  []layout.Candidate `json:"candidates"`
}

type canvasExplainEnvelope struct {
	SchemaVersion int               `json:"schemaVersion"`
	Kind          string            `json:"kind"`
	Board         string            `json:"board"`
	Exists        bool              `json:"exists"`
	ID            string            `json:"id"`
	Pinned        *canvasPointJSON  `json:"pinned"`
	Placement     string            `json:"placement"`
	Applied       bool              `json:"applied"`
	Routing       canvasRoutingJSON `json:"routing"`
}

func newCanvasExplainEnvelope(v *boardView, t *ticket.Ticket, e layout.Explanation) canvasExplainEnvelope {
	env := canvasExplainEnvelope{SchemaVersion: schemaVersion, Kind: "canvas-explain", Board: v.board.Board, Exists: v.exists, ID: t.ID, Applied: true,
		Placement: "status-lanes", Routing: canvasRoutingJSON{Candidates: e.Candidates}}
	if env.Routing.Candidates == nil {
		env.Routing.Candidates = []layout.Candidate{}
	}
	// placement is what the board shows: pinned where somebody put it; by the
	// rules on a board with pens; in status lanes on a board with none.
	if len(env.Routing.Candidates) > 0 {
		env.Placement = "rules"
	}
	if e.Pinned != nil {
		env.Pinned = &canvasPointJSON{X: e.Pinned.X, Y: e.Pinned.Y}
		env.Placement = "pinned"
	}
	if !e.Inbox() {
		d := e.Destination
		env.Routing.Destination = &d
	}
	return env
}

func ids(ts []*ticket.Ticket) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}
